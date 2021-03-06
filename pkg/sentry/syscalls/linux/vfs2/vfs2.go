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

// Package vfs2 provides syscall implementations that use VFS2.
package vfs2

import (
	"gvisor.dev/gvisor/pkg/sentry/syscalls"
	"gvisor.dev/gvisor/pkg/sentry/syscalls/linux"
)

// Override syscall table to add syscalls implementations from this package.
func Override() {
	// Override AMD64.
	s := linux.AMD64
	s.Table[0] = syscalls.Supported("read", Read)
	s.Table[1] = syscalls.Supported("write", Write)
	s.Table[2] = syscalls.Supported("open", Open)
	s.Table[3] = syscalls.Supported("close", Close)
	s.Table[4] = syscalls.Supported("stat", Stat)
	s.Table[5] = syscalls.Supported("fstat", Fstat)
	s.Table[6] = syscalls.Supported("lstat", Lstat)
	s.Table[7] = syscalls.Supported("poll", Poll)
	s.Table[8] = syscalls.Supported("lseek", Lseek)
	s.Table[9] = syscalls.Supported("mmap", Mmap)
	s.Table[16] = syscalls.Supported("ioctl", Ioctl)
	s.Table[17] = syscalls.Supported("pread64", Pread64)
	s.Table[18] = syscalls.Supported("pwrite64", Pwrite64)
	s.Table[19] = syscalls.Supported("readv", Readv)
	s.Table[20] = syscalls.Supported("writev", Writev)
	s.Table[21] = syscalls.Supported("access", Access)
	s.Table[22] = syscalls.Supported("pipe", Pipe)
	s.Table[23] = syscalls.Supported("select", Select)
	s.Table[32] = syscalls.Supported("dup", Dup)
	s.Table[33] = syscalls.Supported("dup2", Dup2)
	delete(s.Table, 40) // sendfile
	s.Table[41] = syscalls.Supported("socket", Socket)
	s.Table[42] = syscalls.Supported("connect", Connect)
	s.Table[43] = syscalls.Supported("accept", Accept)
	s.Table[44] = syscalls.Supported("sendto", SendTo)
	s.Table[45] = syscalls.Supported("recvfrom", RecvFrom)
	s.Table[46] = syscalls.Supported("sendmsg", SendMsg)
	s.Table[47] = syscalls.Supported("recvmsg", RecvMsg)
	s.Table[48] = syscalls.Supported("shutdown", Shutdown)
	s.Table[49] = syscalls.Supported("bind", Bind)
	s.Table[50] = syscalls.Supported("listen", Listen)
	s.Table[51] = syscalls.Supported("getsockname", GetSockName)
	s.Table[52] = syscalls.Supported("getpeername", GetPeerName)
	s.Table[53] = syscalls.Supported("socketpair", SocketPair)
	s.Table[54] = syscalls.Supported("setsockopt", SetSockOpt)
	s.Table[55] = syscalls.Supported("getsockopt", GetSockOpt)
	s.Table[59] = syscalls.Supported("execve", Execve)
	s.Table[72] = syscalls.Supported("fcntl", Fcntl)
	delete(s.Table, 73) // flock
	s.Table[74] = syscalls.Supported("fsync", Fsync)
	s.Table[75] = syscalls.Supported("fdatasync", Fdatasync)
	s.Table[76] = syscalls.Supported("truncate", Truncate)
	s.Table[77] = syscalls.Supported("ftruncate", Ftruncate)
	s.Table[78] = syscalls.Supported("getdents", Getdents)
	s.Table[79] = syscalls.Supported("getcwd", Getcwd)
	s.Table[80] = syscalls.Supported("chdir", Chdir)
	s.Table[81] = syscalls.Supported("fchdir", Fchdir)
	s.Table[82] = syscalls.Supported("rename", Rename)
	s.Table[83] = syscalls.Supported("mkdir", Mkdir)
	s.Table[84] = syscalls.Supported("rmdir", Rmdir)
	s.Table[85] = syscalls.Supported("creat", Creat)
	s.Table[86] = syscalls.Supported("link", Link)
	s.Table[87] = syscalls.Supported("unlink", Unlink)
	s.Table[88] = syscalls.Supported("symlink", Symlink)
	s.Table[89] = syscalls.Supported("readlink", Readlink)
	s.Table[90] = syscalls.Supported("chmod", Chmod)
	s.Table[91] = syscalls.Supported("fchmod", Fchmod)
	s.Table[92] = syscalls.Supported("chown", Chown)
	s.Table[93] = syscalls.Supported("fchown", Fchown)
	s.Table[94] = syscalls.Supported("lchown", Lchown)
	s.Table[132] = syscalls.Supported("utime", Utime)
	s.Table[133] = syscalls.Supported("mknod", Mknod)
	s.Table[137] = syscalls.Supported("statfs", Statfs)
	s.Table[138] = syscalls.Supported("fstatfs", Fstatfs)
	s.Table[161] = syscalls.Supported("chroot", Chroot)
	s.Table[162] = syscalls.Supported("sync", Sync)
	delete(s.Table, 165) // mount
	delete(s.Table, 166) // umount2
	delete(s.Table, 187) // readahead
	s.Table[188] = syscalls.Supported("setxattr", Setxattr)
	s.Table[189] = syscalls.Supported("lsetxattr", Lsetxattr)
	s.Table[190] = syscalls.Supported("fsetxattr", Fsetxattr)
	s.Table[191] = syscalls.Supported("getxattr", Getxattr)
	s.Table[192] = syscalls.Supported("lgetxattr", Lgetxattr)
	s.Table[193] = syscalls.Supported("fgetxattr", Fgetxattr)
	s.Table[194] = syscalls.Supported("listxattr", Listxattr)
	s.Table[195] = syscalls.Supported("llistxattr", Llistxattr)
	s.Table[196] = syscalls.Supported("flistxattr", Flistxattr)
	s.Table[197] = syscalls.Supported("removexattr", Removexattr)
	s.Table[198] = syscalls.Supported("lremovexattr", Lremovexattr)
	s.Table[199] = syscalls.Supported("fremovexattr", Fremovexattr)
	delete(s.Table, 206) // io_setup
	delete(s.Table, 207) // io_destroy
	delete(s.Table, 208) // io_getevents
	delete(s.Table, 209) // io_submit
	delete(s.Table, 210) // io_cancel
	s.Table[213] = syscalls.Supported("epoll_create", EpollCreate)
	s.Table[217] = syscalls.Supported("getdents64", Getdents64)
	delete(s.Table, 221) // fdavise64
	s.Table[232] = syscalls.Supported("epoll_wait", EpollWait)
	s.Table[233] = syscalls.Supported("epoll_ctl", EpollCtl)
	s.Table[235] = syscalls.Supported("utimes", Utimes)
	delete(s.Table, 253) // inotify_init
	delete(s.Table, 254) // inotify_add_watch
	delete(s.Table, 255) // inotify_rm_watch
	s.Table[257] = syscalls.Supported("openat", Openat)
	s.Table[258] = syscalls.Supported("mkdirat", Mkdirat)
	s.Table[259] = syscalls.Supported("mknodat", Mknodat)
	s.Table[260] = syscalls.Supported("fchownat", Fchownat)
	s.Table[261] = syscalls.Supported("futimens", Futimens)
	s.Table[262] = syscalls.Supported("newfstatat", Newfstatat)
	s.Table[263] = syscalls.Supported("unlinkat", Unlinkat)
	s.Table[264] = syscalls.Supported("renameat", Renameat)
	s.Table[265] = syscalls.Supported("linkat", Linkat)
	s.Table[266] = syscalls.Supported("symlinkat", Symlinkat)
	s.Table[267] = syscalls.Supported("readlinkat", Readlinkat)
	s.Table[268] = syscalls.Supported("fchmodat", Fchmodat)
	s.Table[269] = syscalls.Supported("faccessat", Faccessat)
	s.Table[270] = syscalls.Supported("pselect", Pselect)
	s.Table[271] = syscalls.Supported("ppoll", Ppoll)
	delete(s.Table, 275) // splice
	delete(s.Table, 276) // tee
	s.Table[277] = syscalls.Supported("sync_file_range", SyncFileRange)
	s.Table[280] = syscalls.Supported("utimensat", Utimensat)
	s.Table[281] = syscalls.Supported("epoll_pwait", EpollPwait)
	s.Table[282] = syscalls.Supported("signalfd", Signalfd)
	s.Table[283] = syscalls.Supported("timerfd_create", TimerfdCreate)
	s.Table[284] = syscalls.Supported("eventfd", Eventfd)
	delete(s.Table, 285) // fallocate
	s.Table[286] = syscalls.Supported("timerfd_settime", TimerfdSettime)
	s.Table[287] = syscalls.Supported("timerfd_gettime", TimerfdGettime)
	s.Table[288] = syscalls.Supported("accept4", Accept4)
	s.Table[289] = syscalls.Supported("signalfd4", Signalfd4)
	s.Table[290] = syscalls.Supported("eventfd2", Eventfd2)
	s.Table[291] = syscalls.Supported("epoll_create1", EpollCreate1)
	s.Table[292] = syscalls.Supported("dup3", Dup3)
	s.Table[293] = syscalls.Supported("pipe2", Pipe2)
	delete(s.Table, 294) // inotify_init1
	s.Table[295] = syscalls.Supported("preadv", Preadv)
	s.Table[296] = syscalls.Supported("pwritev", Pwritev)
	s.Table[299] = syscalls.Supported("recvmmsg", RecvMMsg)
	s.Table[306] = syscalls.Supported("syncfs", Syncfs)
	s.Table[307] = syscalls.Supported("sendmmsg", SendMMsg)
	s.Table[316] = syscalls.Supported("renameat2", Renameat2)
	delete(s.Table, 319) // memfd_create
	s.Table[322] = syscalls.Supported("execveat", Execveat)
	s.Table[327] = syscalls.Supported("preadv2", Preadv2)
	s.Table[328] = syscalls.Supported("pwritev2", Pwritev2)
	s.Table[332] = syscalls.Supported("statx", Statx)
	s.Init()

	// Override ARM64.
	s = linux.ARM64
	s.Table[63] = syscalls.Supported("read", Read)
	s.Init()
}
