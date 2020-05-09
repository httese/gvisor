// Copyright 2020 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testbench has utilities to send and receive packets and also command
// the DUT to run POSIX functions.
package testbench

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"testing"
	"time"

	"github.com/mohae/deepcopy"
	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/seqnum"
	"gvisor.dev/gvisor/test/packetimpact/netdevs"
)

var (
	// Flags.
	localIPv4  = flag.String("local_ipv4", "", "local IPv4 address for test packets")
	remoteIPv4 = flag.String("remote_ipv4", "", "remote IPv4 address for test packets")
	remoteIPv6 = flag.String("remote_ipv6", "", "remote IPv6 address for test packets")
	remoteMAC  = flag.String("remote_mac", "", "remote mac address for test packets")

	// Pseudo-flags. Filled in based on flag information at flag parse time.
	localIPv6 *string // Local IPv6 address for test packets.
	localMAC  *string // Local mac address for test packets.
)

func genPseudoFlags() error {
	out, err := exec.Command("ip", "addr", "show").Output()
	if err != nil {
		return fmt.Errorf("listing devices: %v", err)
	}
	devs, err := netdevs.ParseDevices(string(out))
	if err != nil {
		return fmt.Errorf("parsing devices: %v", err)
	}

	_, deviceInfo, err := netdevs.FindDeviceByIP(net.ParseIP(*localIPv4), devs)
	if err != nil {
		return fmt.Errorf("can't find deviceInfo: %s", err)
	}

	mac := deviceInfo.MAC.String()
	localMAC = &mac
	v6 := deviceInfo.IPv6Addr.String()
	localIPv6 = &v6

	return nil
}

func portFromSockaddr(sa unix.Sockaddr) (uint16, error) {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return uint16(sa.Port), nil
	case *unix.SockaddrInet6:
		return uint16(sa.Port), nil
	}
	return 0, fmt.Errorf("sockaddr type %T does not contain port", sa)
}

// pickPort makes a new socket and returns the socket FD and port. The domain should be AF_INET or AF_INET6. The caller must close the FD when done with
// the port if there is no error.
func pickPort(domain, typ int) (fd int, sa unix.Sockaddr, err error) {
	fd, err = unix.Socket(domain, typ, 0)
	if err != nil {
		return -1, nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, unix.Close(fd))
		}
	}()
	switch domain {
	case unix.AF_INET:
		var sa4 unix.SockaddrInet4
		copy(sa4.Addr[:], net.ParseIP(*localIPv4).To4())
		sa = &sa4
	case unix.AF_INET6:
		var sa6 unix.SockaddrInet6
		copy(sa6.Addr[:], net.ParseIP(*localIPv6).To16())
		sa = &sa6
	default:
		return -1, nil, fmt.Errorf("invalid domain %d, it should be one of unix.AF_INET or unix.AF_INET6", domain)
	}
	if err = unix.Bind(fd, sa); err != nil {
		return -1, nil, err
	}
	sa, err = unix.Getsockname(fd)
	if err != nil {
		return -1, nil, err
	}
	return fd, sa, nil
}

// layerState stores the state of a layer of a connection.
type layerState interface {
	// outgoing returns an outgoing layer to be sent in a frame. It should not
	// update layerState, that is done in layerState.sent.
	outgoing() Layer

	// incoming creates an expected Layer for comparing against a received Layer.
	// Because the expectation can depend on values in the received Layer, it is
	// an input to incoming. For example, the ACK number needs to be checked in a
	// TCP packet but only if the ACK flag is set in the received packet. It
	// should not update layerState, that is done in layerState.received. The
	// caller takes ownership of the returned Layer.
	incoming(received Layer) Layer

	// sent updates the layerState based on the Layer that was sent. The input is
	// a Layer with all prev and next pointers populated so that the entire frame
	// as it was sent is available.
	sent(sent Layer) error

	// received updates the layerState based on a Layer that is receieved. The
	// input is a Layer with all prev and next pointers populated so that the
	// entire frame as it was receieved is available.
	received(received Layer) error

	// close frees associated resources held by the LayerState.
	close() error
}

// etherState maintains state about an Ethernet connection.
type etherState struct {
	out, in Ether
}

var _ layerState = (*etherState)(nil)

// newEtherState creates a new etherState.
func newEtherState(out, in Ether) (*etherState, error) {
	lMAC, err := tcpip.ParseMACAddress(*localMAC)
	if err != nil {
		return nil, err
	}

	rMAC, err := tcpip.ParseMACAddress(*remoteMAC)
	if err != nil {
		return nil, err
	}
	s := etherState{
		out: Ether{SrcAddr: &lMAC, DstAddr: &rMAC},
		in:  Ether{SrcAddr: &rMAC, DstAddr: &lMAC},
	}
	if err := s.out.merge(&out); err != nil {
		return nil, err
	}
	if err := s.in.merge(&in); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *etherState) outgoing() Layer {
	return deepcopy.Copy(&s.out).(Layer)
}

// incoming implements layerState.incoming.
func (s *etherState) incoming(Layer) Layer {
	return deepcopy.Copy(&s.in).(Layer)
}

func (*etherState) sent(Layer) error {
	return nil
}

func (*etherState) received(Layer) error {
	return nil
}

func (*etherState) close() error {
	return nil
}

// ipv4State maintains state about an IPv4 connection.
type ipv4State struct {
	out, in IPv4
}

var _ layerState = (*ipv4State)(nil)

// newIPv4State creates a new ipv4State.
func newIPv4State(out, in IPv4) (*ipv4State, error) {
	lIP := tcpip.Address(net.ParseIP(*localIPv4).To4())
	rIP := tcpip.Address(net.ParseIP(*remoteIPv4).To4())
	s := ipv4State{
		out: IPv4{SrcAddr: &lIP, DstAddr: &rIP},
		in:  IPv4{SrcAddr: &rIP, DstAddr: &lIP},
	}
	if err := s.out.merge(&out); err != nil {
		return nil, err
	}
	if err := s.in.merge(&in); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *ipv4State) outgoing() Layer {
	return deepcopy.Copy(&s.out).(Layer)
}

// incoming implements layerState.incoming.
func (s *ipv4State) incoming(Layer) Layer {
	return deepcopy.Copy(&s.in).(Layer)
}

func (*ipv4State) sent(Layer) error {
	return nil
}

func (*ipv4State) received(Layer) error {
	return nil
}

func (*ipv4State) close() error {
	return nil
}

// ipv6State maintains state about an IPv6 connection.
type ipv6State struct {
	out, in IPv6
}

var _ layerState = (*ipv6State)(nil)

// newIPv6State creates a new ipv6State.
func newIPv6State(out, in IPv6) (*ipv6State, error) {
	lIP := tcpip.Address(net.ParseIP(*localIPv6).To16())
	rIP := tcpip.Address(net.ParseIP(*remoteIPv6).To16())
	s := ipv6State{
		out: IPv6{SrcAddr: &lIP, DstAddr: &rIP},
		in:  IPv6{SrcAddr: &rIP, DstAddr: &lIP},
	}
	if err := s.out.merge(&out); err != nil {
		return nil, err
	}
	if err := s.in.merge(&in); err != nil {
		return nil, err
	}
	return &s, nil
}

// outgoing returns an outgoing layer to be sent in a frame.
func (s *ipv6State) outgoing() Layer {
	return deepcopy.Copy(&s.out).(Layer)
}

func (s *ipv6State) incoming(Layer) Layer {
	return deepcopy.Copy(&s.in).(Layer)
}

func (s *ipv6State) sent(Layer) error {
	// Nothing to do.
	return nil
}

func (s *ipv6State) received(Layer) error {
	// Nothing to do.
	return nil
}

// close cleans up any resources held.
func (s *ipv6State) close() error {
	return nil
}

// tcpState maintains state about a TCP connection.
type tcpState struct {
	out, in                   TCP
	localSeqNum, remoteSeqNum *seqnum.Value
	synAck                    *TCP
	portPickerFD              int
	finSent                   bool
}

var _ layerState = (*tcpState)(nil)

// SeqNumValue is a helper routine that allocates a new seqnum.Value value to
// store v and returns a pointer to it.
func SeqNumValue(v seqnum.Value) *seqnum.Value {
	return &v
}

// newTCPState creates a new TCPState.
func newTCPState(domain int, out, in TCP) (*tcpState, error) {
	portPickerFD, localAddr, err := pickPort(domain, unix.SOCK_STREAM)
	if err != nil {
		return nil, err
	}
	localPort, err := portFromSockaddr(localAddr)
	if err != nil {
		return nil, err
	}
	s := tcpState{
		out:          TCP{SrcPort: &localPort},
		in:           TCP{DstPort: &localPort},
		localSeqNum:  SeqNumValue(seqnum.Value(rand.Uint32())),
		portPickerFD: portPickerFD,
		finSent:      false,
	}
	if err := s.out.merge(&out); err != nil {
		return nil, err
	}
	if err := s.in.merge(&in); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *tcpState) outgoing() Layer {
	newOutgoing := deepcopy.Copy(s.out).(TCP)
	if s.localSeqNum != nil {
		newOutgoing.SeqNum = Uint32(uint32(*s.localSeqNum))
	}
	if s.remoteSeqNum != nil {
		newOutgoing.AckNum = Uint32(uint32(*s.remoteSeqNum))
	}
	return &newOutgoing
}

// incoming implements layerState.incoming.
func (s *tcpState) incoming(received Layer) Layer {
	tcpReceived, ok := received.(*TCP)
	if !ok {
		return nil
	}
	newIn := deepcopy.Copy(s.in).(TCP)
	if s.remoteSeqNum != nil {
		newIn.SeqNum = Uint32(uint32(*s.remoteSeqNum))
	}
	if s.localSeqNum != nil && (*tcpReceived.Flags&header.TCPFlagAck) != 0 {
		// The caller didn't specify an AckNum so we'll expect the calculated one,
		// but only if the ACK flag is set because the AckNum is not valid in a
		// header if ACK is not set.
		newIn.AckNum = Uint32(uint32(*s.localSeqNum))
	}
	return &newIn
}

func (s *tcpState) sent(sent Layer) error {
	tcp, ok := sent.(*TCP)
	if !ok {
		return fmt.Errorf("can't update tcpState with %T Layer", sent)
	}
	if !s.finSent {
		// update localSeqNum by the payload only when FIN is not yet sent by us
		for current := tcp.next(); current != nil; current = current.next() {
			s.localSeqNum.UpdateForward(seqnum.Size(current.length()))
		}
	}
	if tcp.Flags != nil && *tcp.Flags&(header.TCPFlagSyn|header.TCPFlagFin) != 0 {
		s.localSeqNum.UpdateForward(1)
	}
	if *tcp.Flags&(header.TCPFlagFin) != 0 {
		s.finSent = true
	}
	return nil
}

func (s *tcpState) received(l Layer) error {
	tcp, ok := l.(*TCP)
	if !ok {
		return fmt.Errorf("can't update tcpState with %T Layer", l)
	}
	s.remoteSeqNum = SeqNumValue(seqnum.Value(*tcp.SeqNum))
	if *tcp.Flags&(header.TCPFlagSyn|header.TCPFlagFin) != 0 {
		s.remoteSeqNum.UpdateForward(1)
	}
	for current := tcp.next(); current != nil; current = current.next() {
		s.remoteSeqNum.UpdateForward(seqnum.Size(current.length()))
	}
	return nil
}

// close frees the port associated with this connection.
func (s *tcpState) close() error {
	if err := unix.Close(s.portPickerFD); err != nil {
		return err
	}
	s.portPickerFD = -1
	return nil
}

// udpState maintains state about a UDP connection.
type udpState struct {
	out, in      UDP
	portPickerFD int
}

var _ layerState = (*udpState)(nil)

// newUDPState creates a new udpState.
func newUDPState(domain int, out, in UDP) (*udpState, unix.Sockaddr, error) {
	portPickerFD, localAddr, err := pickPort(domain, unix.SOCK_DGRAM)
	if err != nil {
		return nil, nil, err
	}
	localPort, err := portFromSockaddr(localAddr)
	if err != nil {
		return nil, nil, err
	}
	s := udpState{
		out:          UDP{SrcPort: &localPort},
		in:           UDP{DstPort: &localPort},
		portPickerFD: portPickerFD,
	}
	if err := s.out.merge(&out); err != nil {
		return nil, nil, err
	}
	if err := s.in.merge(&in); err != nil {
		return nil, nil, err
	}
	return &s, localAddr, nil
}

func (s *udpState) outgoing() Layer {
	return deepcopy.Copy(&s.out).(Layer)
}

// incoming implements layerState.incoming.
func (s *udpState) incoming(Layer) Layer {
	return deepcopy.Copy(&s.in).(Layer)
}

func (*udpState) sent(l Layer) error {
	return nil
}

func (*udpState) received(l Layer) error {
	return nil
}

// close frees the port associated with this connection.
func (s *udpState) close() error {
	if err := unix.Close(s.portPickerFD); err != nil {
		return err
	}
	s.portPickerFD = -1
	return nil
}

// Connection holds a collection of layer states for maintaining a connection
// along with sockets for sniffer and injecting packets.
type Connection struct {
	layerStates []layerState
	injector    Injector
	sniffer     Sniffer
	localAddr   unix.Sockaddr
	t           *testing.T
}

// Returns the default incoming frame against which to match. If received is
// longer than layerStates then that may still count as a match. The reverse is
// never a match and nil is returned.
func (conn *Connection) incoming(received Layers) Layers {
	if len(received) < len(conn.layerStates) {
		return nil
	}
	in := Layers{}
	for i, s := range conn.layerStates {
		toMatch := s.incoming(received[i])
		if toMatch == nil {
			return nil
		}
		in = append(in, toMatch)
	}
	return in
}

func (conn *Connection) match(override, received Layers) bool {
	toMatch := conn.incoming(received)
	if toMatch == nil {
		return false // Not enough layers in gotLayers for matching.
	}
	if err := toMatch.merge(override); err != nil {
		return false // Failing to merge is not matching.
	}
	return toMatch.match(received)
}

// Close frees associated resources held by the Connection.
func (conn *Connection) Close() {
	errs := multierr.Combine(conn.sniffer.close(), conn.injector.close())
	for _, s := range conn.layerStates {
		if err := s.close(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("unable to close %+v: %s", s, err))
		}
	}
	if errs != nil {
		conn.t.Fatalf("unable to close %+v: %s", conn, errs)
	}
}

// CreateFrame builds a frame for the connection with layer overriding defaults
// of the innermost layer and additionalLayers added after it.
func (conn *Connection) CreateFrame(layer Layer, additionalLayers ...Layer) Layers {
	var layersToSend Layers
	for _, s := range conn.layerStates {
		layersToSend = append(layersToSend, s.outgoing())
	}
	if err := layersToSend[len(layersToSend)-1].merge(layer); err != nil {
		conn.t.Fatalf("can't merge %+v into %+v: %s", layer, layersToSend[len(layersToSend)-1], err)
	}
	layersToSend = append(layersToSend, additionalLayers...)
	return layersToSend
}

// SendFrame sends a frame on the wire and updates the state of all layers.
func (conn *Connection) SendFrame(frame Layers) {
	outBytes, err := frame.ToBytes()
	if err != nil {
		conn.t.Fatalf("can't build outgoing packet: %s", err)
	}
	conn.injector.Send(outBytes)

	// frame might have nil values where the caller wanted to use default values.
	// sentFrame will have no nil values in it because it comes from parsing the
	// bytes that were actually sent.
	sentFrame := parse(parseEther, outBytes)
	// Update the state of each layer based on what was sent.
	for i, s := range conn.layerStates {
		if err := s.sent(sentFrame[i]); err != nil {
			conn.t.Fatalf("Unable to update the state of %+v with %s: %s", s, sentFrame[i], err)
		}
	}
}

// Send a packet with reasonable defaults. Potentially override the final layer
// in the connection with the provided layer and add additionLayers.
func (conn *Connection) Send(layer Layer, additionalLayers ...Layer) {
	conn.SendFrame(conn.CreateFrame(layer, additionalLayers...))
}

// recvFrame gets the next successfully parsed frame (of type Layers) within the
// timeout provided. If no parsable frame arrives before the timeout, it returns
// nil.
func (conn *Connection) recvFrame(timeout time.Duration) Layers {
	if timeout <= 0 {
		return nil
	}
	b := conn.sniffer.Recv(timeout)
	if b == nil {
		return nil
	}
	return parse(parseEther, b)
}

// layersError stores the Layers that we got and the Layers that we wanted to
// match.
type layersError struct {
	got, want Layers
}

func (e *layersError) Error() string {
	return e.got.diff(e.want)
}

// Expect expects a frame with the final layerStates layer matching the
// provided Layer within the timeout specified. If it doesn't arrive in time,
// an error is returned.
func (conn *Connection) Expect(layer Layer, timeout time.Duration) (Layer, error) {
	// Make a frame that will ignore all but the final layer.
	layers := make([]Layer, len(conn.layerStates))
	layers[len(layers)-1] = layer

	gotFrame, err := conn.ExpectFrame(layers, timeout)
	if err != nil {
		return nil, err
	}
	if len(conn.layerStates)-1 < len(gotFrame) {
		return gotFrame[len(conn.layerStates)-1], nil
	}
	conn.t.Fatal("the received frame should be at least as long as the expected layers")
	panic("unreachable")
}

// ExpectFrame expects a frame that matches the provided Layers within the
// timeout specified. If one arrives in time, the Layers is returned without an
// error. If it doesn't arrive in time, it returns nil and error is non-nil.
func (conn *Connection) ExpectFrame(layers Layers, timeout time.Duration) (Layers, error) {
	deadline := time.Now().Add(timeout)
	var errs error
	for {
		var gotLayers Layers
		if timeout = time.Until(deadline); timeout > 0 {
			gotLayers = conn.recvFrame(timeout)
		}
		if gotLayers == nil {
			if errs == nil {
				return nil, fmt.Errorf("got no frames matching %v during %s", layers, timeout)
			}
			return nil, fmt.Errorf("got no frames matching %v during %s: got %w", layers, timeout, errs)
		}
		if conn.match(layers, gotLayers) {
			for i, s := range conn.layerStates {
				if err := s.received(gotLayers[i]); err != nil {
					conn.t.Fatal(err)
				}
			}
			return gotLayers, nil
		}
		errs = multierr.Combine(errs, &layersError{got: gotLayers, want: conn.incoming(gotLayers)})
	}
}

// Drain drains the sniffer's receive buffer by receiving packets until there's
// nothing else to receive.
func (conn *Connection) Drain() {
	conn.sniffer.Drain()
}

// TCPIPv4 maintains the state for all the layers in a TCP/IPv4 connection.
type TCPIPv4 Connection

// NewTCPIPv4 creates a new TCPIPv4 connection with reasonable defaults.
func NewTCPIPv4(t *testing.T, outgoingTCP, incomingTCP TCP) TCPIPv4 {
	etherState, err := newEtherState(Ether{}, Ether{})
	if err != nil {
		t.Fatalf("can't make etherState: %s", err)
	}
	ipv4State, err := newIPv4State(IPv4{}, IPv4{})
	if err != nil {
		t.Fatalf("can't make ipv4State: %s", err)
	}
	tcpState, err := newTCPState(unix.AF_INET, outgoingTCP, incomingTCP)
	if err != nil {
		t.Fatalf("can't make tcpState: %s", err)
	}
	injector, err := NewInjector(t)
	if err != nil {
		t.Fatalf("can't make injector: %s", err)
	}
	sniffer, err := NewSniffer(t)
	if err != nil {
		t.Fatalf("can't make sniffer: %s", err)
	}

	return TCPIPv4{
		layerStates: []layerState{etherState, ipv4State, tcpState},
		injector:    injector,
		sniffer:     sniffer,
		t:           t,
	}
}

// Handshake performs a TCP 3-way handshake. The input Connection should have a
// final TCP Layer.
func (conn *TCPIPv4) Handshake() {
	// Send the SYN.
	conn.Send(TCP{Flags: Uint8(header.TCPFlagSyn)})

	// Wait for the SYN-ACK.
	synAck, err := conn.Expect(TCP{Flags: Uint8(header.TCPFlagSyn | header.TCPFlagAck)}, time.Second)
	if synAck == nil {
		conn.t.Fatalf("didn't get synack during handshake: %s", err)
	}
	conn.layerStates[len(conn.layerStates)-1].(*tcpState).synAck = synAck

	// Send an ACK.
	conn.Send(TCP{Flags: Uint8(header.TCPFlagAck)})
}

// ExpectData is a convenient method that expects a Layer and the Layer after
// it. If it doens't arrive in time, it returns nil.
func (conn *TCPIPv4) ExpectData(tcp *TCP, payload *Payload, timeout time.Duration) (Layers, error) {
	expected := make([]Layer, len(conn.layerStates))
	expected[len(expected)-1] = tcp
	if payload != nil {
		expected = append(expected, payload)
	}
	return (*Connection)(conn).ExpectFrame(expected, timeout)
}

// Send a packet with reasonable defaults. Potentially override the TCP layer in
// the connection with the provided layer and add additionLayers.
func (conn *TCPIPv4) Send(tcp TCP, additionalLayers ...Layer) {
	(*Connection)(conn).Send(&tcp, additionalLayers...)
}

// Close frees associated resources held by the TCPIPv4 connection.
func (conn *TCPIPv4) Close() {
	(*Connection)(conn).Close()
}

// Expect expects a frame with the TCP layer matching the provided TCP within
// the timeout specified. If it doesn't arrive in time, an error is returned.
func (conn *TCPIPv4) Expect(tcp TCP, timeout time.Duration) (*TCP, error) {
	layer, err := (*Connection)(conn).Expect(&tcp, timeout)
	if layer == nil {
		return nil, err
	}
	gotTCP, ok := layer.(*TCP)
	if !ok {
		conn.t.Fatalf("expected %s to be TCP", layer)
	}
	return gotTCP, err
}

func (conn *TCPIPv4) state() *tcpState {
	state, ok := conn.layerStates[len(conn.layerStates)-1].(*tcpState)
	if !ok {
		conn.t.Fatalf("expected final state of %v to be tcpState", conn.layerStates)
	}
	return state
}

// RemoteSeqNum returns the next expected sequence number from the DUT.
func (conn *TCPIPv4) RemoteSeqNum() *seqnum.Value {
	return conn.state().remoteSeqNum
}

// LocalSeqNum returns the next sequence number to send from the testbench.
func (conn *TCPIPv4) LocalSeqNum() *seqnum.Value {
	return conn.state().localSeqNum
}

// SynAck returns the SynAck that was part of the handshake.
func (conn *TCPIPv4) SynAck() *TCP {
	return conn.state().synAck
}

// IPv6Conn maintains the state for all the layers in a IPv6 connection.
type IPv6Conn Connection

// NewIPv6Conn creates a new IPv6Conn connection with reasonable defaults.
func NewIPv6Conn(t *testing.T, outgoingIPv6, incomingIPv6 IPv6) IPv6Conn {
	etherState, err := newEtherState(Ether{}, Ether{})
	if err != nil {
		t.Fatalf("can't make EtherState: %s", err)
	}
	ipv6State, err := newIPv6State(outgoingIPv6, incomingIPv6)
	if err != nil {
		t.Fatalf("can't make IPv6State: %s", err)
	}

	injector, err := NewInjector(t)
	if err != nil {
		t.Fatalf("can't make injector: %s", err)
	}
	sniffer, err := NewSniffer(t)
	if err != nil {
		t.Fatalf("can't make sniffer: %s", err)
	}

	return IPv6Conn{
		layerStates: []layerState{etherState, ipv6State},
		injector:    injector,
		sniffer:     sniffer,
		t:           t,
	}
}

// SendFrame sends a frame on the wire and updates the state of all layers.
func (conn *IPv6Conn) SendFrame(frame Layers) {
	(*Connection)(conn).SendFrame(frame)
}

// CreateFrame builds a frame for the connection with ipv6 overriding the ipv6
// layer defaults and additionalLayers added after it.
func (conn *IPv6Conn) CreateFrame(ipv6 IPv6, additionalLayers ...Layer) Layers {
	return (*Connection)(conn).CreateFrame(&ipv6, additionalLayers...)
}

// Close to clean up any resources held.
func (conn *IPv6Conn) Close() {
	(*Connection)(conn).Close()
}

// ExpectFrame expects a frame that matches the provided Layers within the
// timeout specified. If it doesn't arrive in time, an error is returned.
func (conn *IPv6Conn) ExpectFrame(frame Layers, timeout time.Duration) (Layers, error) {
	return (*Connection)(conn).ExpectFrame(frame, timeout)
}

// Drain drains the sniffer's receive buffer by receiving packets until there's
// nothing else to receive.
func (conn *TCPIPv4) Drain() {
	conn.sniffer.Drain()
}

// UDPIPv4 maintains the state for all the layers in a UDP/IPv4 connection.
type UDPIPv4 Connection

// NewUDPIPv4 creates a new UDPIPv4 connection with reasonable defaults.
func NewUDPIPv4(t *testing.T, outgoingUDP, incomingUDP UDP) UDPIPv4 {
	etherState, err := newEtherState(Ether{}, Ether{})
	if err != nil {
		t.Fatalf("can't make etherState: %s", err)
	}
	ipv4State, err := newIPv4State(IPv4{}, IPv4{})
	if err != nil {
		t.Fatalf("can't make ipv4State: %s", err)
	}
	udpState, localAddr, err := newUDPState(unix.AF_INET, outgoingUDP, incomingUDP)
	if err != nil {
		t.Fatalf("can't make udpState: %s", err)
	}
	injector, err := NewInjector(t)
	if err != nil {
		t.Fatalf("can't make injector: %s", err)
	}
	sniffer, err := NewSniffer(t)
	if err != nil {
		t.Fatalf("can't make sniffer: %s", err)
	}

	return UDPIPv4{
		layerStates: []layerState{etherState, ipv4State, udpState},
		injector:    injector,
		sniffer:     sniffer,
		localAddr:   localAddr,
		t:           t,
	}
}

// LocalAddr gets the local socket address of this connection.
func (conn *UDPIPv4) LocalAddr() unix.Sockaddr {
	return conn.localAddr
}

// CreateFrame builds a frame for the connection with layer overriding defaults
// of the innermost layer and additionalLayers added after it.
func (conn *UDPIPv4) CreateFrame(layer Layer, additionalLayers ...Layer) Layers {
	return (*Connection)(conn).CreateFrame(layer, additionalLayers...)
}

// Send a packet with reasonable defaults. Potentially override the UDP layer in
// the connection with the provided layer and add additionLayers.
func (conn *UDPIPv4) Send(udp UDP, additionalLayers ...Layer) {
	(*Connection)(conn).Send(&udp, additionalLayers...)
}

// SendFrame sends a frame on the wire and updates the state of all layers.
func (conn *UDPIPv4) SendFrame(frame Layers) {
	(*Connection)(conn).SendFrame(frame)
}

// SendIP sends a packet with additionalLayers following the IP layer in the
// connection.
func (conn *UDPIPv4) SendIP(additionalLayers ...Layer) {
	var layersToSend Layers
	for _, s := range conn.layerStates[:len(conn.layerStates)-1] {
		layersToSend = append(layersToSend, s.outgoing())
	}
	layersToSend = append(layersToSend, additionalLayers...)
	conn.SendFrame(layersToSend)
}

// Expect expects a frame with the UDP layer matching the provided UDP within
// the timeout specified. If it doesn't arrive in time, an error is returned.
func (conn *UDPIPv4) Expect(udp UDP, timeout time.Duration) (*UDP, error) {
	conn.t.Helper()
	layer, err := (*Connection)(conn).Expect(&udp, timeout)
	if layer == nil {
		return nil, err
	}
	gotUDP, ok := layer.(*UDP)
	if !ok {
		conn.t.Fatalf("expected %s to be UDP", layer)
	}
	return gotUDP, err
}

// ExpectData is a convenient method that expects a Layer and the Layer after
// it. If it doens't arrive in time, it returns nil.
func (conn *UDPIPv4) ExpectData(udp UDP, payload Payload, timeout time.Duration) (Layers, error) {
	conn.t.Helper()
	expected := make([]Layer, len(conn.layerStates))
	expected[len(expected)-1] = &udp
	if payload.length() != 0 {
		expected = append(expected, &payload)
	}
	return (*Connection)(conn).ExpectFrame(expected, timeout)
}

// Close frees associated resources held by the UDPIPv4 connection.
func (conn *UDPIPv4) Close() {
	(*Connection)(conn).Close()
}

// Drain drains the sniffer's receive buffer by receiving packets until there's
// nothing else to receive.
func (conn *UDPIPv4) Drain() {
	conn.sniffer.Drain()
}
