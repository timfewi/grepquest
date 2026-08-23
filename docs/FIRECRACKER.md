# Firecracker production boundary

The in-process Firecracker backend is an integration and trusted-host
development launcher. It verifies complete artifact digests, configures no NIC
or host filesystem share, and exposes only the bounded guest-agent protocol,
but it does not invoke the Firecracker jailer. Do not expose it directly as a
public multi-user service.

## Development acceptance test

`just vm-build` builds a focused Linux guest kernel, initrd, and read-only
rootfs. `just vm-test` additionally boots those artifacts on `/dev/kvm`, waits
for an explicit guest-agent readiness handshake, installs a challenge over
vsock, drops the grep process to UID/GID 1000, and verifies its result. This
end-to-end check is required after changing the kernel, images, protocol, agent,
or Firecracker backend, but it does not exercise the production jailer.

## Required production supervisor

A production deployment must launch every microVM through a privileged,
reviewed supervisor that owns the host boundary. The supervisor must:

- use matching statically linked Firecracker and jailer binaries;
- assign a dedicated non-root UID/GID to each isolation domain;
- use a root-owned jail base and trusted, digest-addressed guest artifacts;
- create a cgroup-v2 leaf with explicit `memory.max`, `pids.max`, and `cpu.max`;
- use a new PID namespace and bounded file-descriptor/file-size limits;
- provide no network namespace or TAP device unless the product explicitly
  requires and reviews networking;
- clean up jail directories and cgroups after every process exit; and
- keep swap disabled where guest-memory remanence is in scope.

Run the read-only host audit before starting a production supervisor:

```console
export GREPQUEST_JAIL_BASE=/srv/jailer
export GREPQUEST_CGROUP=grepquest/example
export GREPQUEST_FIRECRACKER_UID=1234
export GREPQUEST_FIRECRACKER_GID=1234
just vm-host-check
```

The audit fails closed when a path is missing or writable by unprivileged
users, the binaries differ in version, KVM is inaccessible, a cgroup limit is
unbounded, or swap is enabled. It never creates users, directories, cgroups,
permissions, or VMs. Passing the audit is evidence only for host prerequisites;
it is not evidence that the current in-process launcher is production-safe.

The upstream Firecracker jailer treats its input paths and operator as trusted.
Follow the version-matched upstream jailer and production-host documentation
when implementing the external supervisor.
