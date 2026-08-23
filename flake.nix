{
  description = "GrepQuest — learn GNU grep through deterministic terminal challenges";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      version = nixpkgs.lib.removeSuffix "\n" (builtins.readFile ./internal/buildinfo/VERSION);
      suiVersion = "1.78.0";
      suiRelease = "testnet-v${suiVersion}";
      suiAssets = {
        x86_64-linux = {
          archiveArch = "x86_64";
          hash = "sha256-hX17IGK5GL+3uyhx5S7WEsExrHmistQXGrYJCoMEbF8=";
        };
        aarch64-linux = {
          archiveArch = "aarch64";
          hash = "sha256-o45N/k1LWDKZbSwux7CbgMW1MFjfxlWuqSgtrF0jwHQ=";
        };
      };
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
      ];
      mkSuiCli =
        pkgs:
        let
          asset = suiAssets.${pkgs.stdenv.hostPlatform.system};
        in
        pkgs.stdenvNoCC.mkDerivation {
          pname = "sui";
          version = suiVersion;
          src = pkgs.fetchurl {
            url = "https://github.com/MystenLabs/sui/releases/download/${suiRelease}/sui-${suiRelease}-ubuntu-${asset.archiveArch}.tgz";
            inherit (asset) hash;
          };
          sourceRoot = ".";
          nativeBuildInputs = [ pkgs.autoPatchelfHook ];
          buildInputs = [
            pkgs.stdenv.cc.cc.lib
            pkgs.openssl
            pkgs.zlib
          ];
          unpackPhase = ''
            runHook preUnpack
            tar -xzf "$src" ./sui
            runHook postUnpack
          '';
          installPhase = ''
            runHook preInstall
            install -Dm755 sui "$out/bin/sui"
            runHook postInstall
          '';
          meta = {
            description = "Sui command-line interface";
            homepage = "https://github.com/MystenLabs/sui";
            license = pkgs.lib.licenses.asl20;
            mainProgram = "sui";
            platforms = builtins.attrNames suiAssets;
            sourceProvenance = [ pkgs.lib.sourceTypes.binaryNativeCode ];
          };
        };
      mkAsciiMotionMcp =
        pkgs:
        pkgs.buildNpmPackage {
          pname = "ascii-motion-mcp";
          version = "2.2.0";
          src = ./tools/ascii-motion-mcp;
          npmDepsHash = "sha256-jS/R2jh7FHsmDVPgsv/LiFGc1BnK4NX7DIf0O6CW/JU=";
          dontNpmBuild = true;
          nativeBuildInputs = [ pkgs.makeWrapper ];
          installPhase = ''
            runHook preInstall

            mkdir -p "$out/bin" "$out/libexec/ascii-motion-mcp"
            cp -R node_modules "$out/libexec/ascii-motion-mcp/"
            makeWrapper ${pkgs.nodejs_22}/bin/node "$out/bin/ascii-motion-mcp" \
              --add-flags "$out/libexec/ascii-motion-mcp/node_modules/ascii-motion-mcp/dist/index.js"

            runHook postInstall
          '';
          meta = {
            description = "MCP server for creating and exporting ASCII Motion projects";
            homepage = "https://github.com/CameronFoxly/ascii-motion-mcp";
            license = pkgs.lib.licenses.mit;
            mainProgram = "ascii-motion-mcp";
          };
        };
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          suiCli = mkSuiCli pkgs;
          asciiMotionMcp = mkAsciiMotionMcp pkgs;
          guestLinux = pkgs.linuxPackages.kernel.override {
            defconfig = "tinyconfig";
            enableCommonConfig = false;
            autoModules = false;
            preferBuiltin = true;
            ignoreConfigErrors = false;
            structuredExtraConfig = with pkgs.lib.kernel; {
              BINFMT_ELF = yes;
              BINFMT_SCRIPT = yes;
              BLOCK = yes;
              BLK_DEV_INITRD = yes;
              DEVTMPFS = yes;
              DEVTMPFS_MOUNT = yes;
              EPOLL = yes;
              EVENTFD = yes;
              EXT4_FS = yes;
              FUTEX = yes;
              HYPERVISOR_GUEST = yes;
              INOTIFY_USER = yes;
              KVM_GUEST = yes;
              MODULES = yes;
              MULTIUSER = yes;
              NET = yes;
              PARAVIRT = yes;
              PRINTK = yes;
              PROC_FS = yes;
              RD_GZIP = yes;
              SERIAL_8250 = yes;
              SERIAL_8250_CONSOLE = yes;
              SHMEM = yes;
              SYSFS = yes;
              TIMERFD = yes;
              TMPFS = yes;
              TTY = yes;
              UNIX = yes;
              VIRTIO_MENU = yes;
              VIRTIO_MMIO = yes;
              VIRTIO_MMIO_CMDLINE_DEVICES = yes;
              VIRTIO_BLK = yes;
              VSOCKETS = yes;
              VIRTIO_VSOCKETS = yes;
            };
          };
          grepquest = pkgs.buildGo125Module {
            pname = "grepquest";
            inherit version;
            src = self;
            vendorHash = "sha256-oqDToT4mLze2Onc1KaWh+SSMz1OTVdu0e0jCFbPTPqA=";
            subPackages = [ "cmd/grepquest" ];
            ldflags = [
              "-s"
              "-w"
            ];
            meta = {
              description = "Local-first terminal roguelite for learning GNU grep";
              homepage = "https://github.com/timfewi/grepquest";
              license = pkgs.lib.licenses.mit;
              mainProgram = "grepquest";
              platforms = pkgs.lib.platforms.linux;
            };
          };
          guestAgent = pkgs.buildGo125Module {
            pname = "grepquest-agent";
            inherit version;
            src = self;
            vendorHash = "sha256-oqDToT4mLze2Onc1KaWh+SSMz1OTVdu0e0jCFbPTPqA=";
            subPackages = [ "guest/agent" ];
            env.CGO_ENABLED = "0";
            ldflags = [
              "-s"
              "-w"
            ];
          };
        in
        {
          default = grepquest;
          sui = suiCli;
          ascii-motion-mcp = asciiMotionMcp;
          inherit grepquest guestAgent;
        }
        // pkgs.lib.optionalAttrs (system == "x86_64-linux") (
          let
            guestRootfs = pkgs.callPackage "${nixpkgs}/nixos/lib/make-ext4-fs.nix" {
              storePaths = [
                guestAgent
                pkgs.gnugrep
                pkgs.busybox
              ];
              volumeLabel = "grepquest";
              populateImageCommands = ''
                mkdir -p files/bin files/sbin files/etc files/proc files/sys files/dev files/run
                ln -s ${guestAgent}/bin/agent files/bin/grepquest-agent
                ln -s ${pkgs.gnugrep}/bin/grep files/bin/grep
                ln -s ${pkgs.busybox}/bin/busybox files/bin/busybox
                ln -s /bin/busybox files/bin/sh
                ln -s /bin/busybox files/bin/chown
                ln -s /bin/busybox files/bin/mkdir
                ln -s /bin/busybox files/bin/mount
                cp ${./guest/init/init.sh} files/sbin/init
                chmod 0555 files/sbin/init
                printf 'root:x:0:0:root:/root:/bin/sh\nplayer:x:1000:1000:player:/run/grepquest/player:/bin/sh\n' > files/etc/passwd
                printf 'root:x:0:\nplayer:x:1000:\n' > files/etc/group
              '';
            };
            guestKernel =
              pkgs.runCommand "grepquest-firecracker-kernel" { nativeBuildInputs = [ pkgs.binutils ]; }
                ''
                  objcopy --strip-debug ${guestLinux.dev}/vmlinux $out
                '';
            stage1Init = pkgs.writeScript "grepquest-stage1-init" (builtins.readFile ./guest/init/stage1.sh);
            guestInitrd = pkgs.makeInitrd {
              compressor = "gzip";
              contents = [
                {
                  object = stage1Init;
                  symlink = "/init";
                }
                {
                  object = "${pkgs.busybox}/bin/busybox";
                  symlink = "/bin/busybox";
                }
                {
                  object = "${pkgs.busybox}/bin/busybox";
                  symlink = "/bin/sh";
                }
                {
                  object = "${pkgs.busybox}/bin/busybox";
                  symlink = "/bin/mount";
                }
                {
                  object = "${pkgs.busybox}/bin/busybox";
                  symlink = "/bin/mkdir";
                }
                {
                  object = "${pkgs.busybox}/bin/busybox";
                  symlink = "/bin/switch_root";
                }
              ];
            };
          in
          {
            inherit guestRootfs guestKernel guestInitrd;
          }
        )
      );
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          suiCli = mkSuiCli pkgs;
          asciiMotionMcp = mkAsciiMotionMcp pkgs;
          grepquestAsciiMotion = pkgs.writeShellApplication {
            name = "grepquest-ascii-motion";
            runtimeInputs = [
              asciiMotionMcp
              pkgs.bubblewrap
              pkgs.coreutils
              pkgs.git
            ];
            text = builtins.readFile ./tools/ascii-motion-mcp/grepquest-ascii-motion.sh;
          };
          suiDevKeystore = pkgs.writeText "grepquest-sui-keystore" ''
            []
          '';
          suiDevClientConfigTemplate = pkgs.writeText "grepquest-sui-client.yaml.in" ''
            keystore:
              File: @SUI_CONFIG_DIR@/sui.keystore
            external_keys: null
            envs:
              - alias: testnet
                rpc: "https://fullnode.testnet.sui.io:443"
                ws: null
                basic_auth: null
            active_env: testnet
            active_address: null
          '';
          suiDevConfigInit = pkgs.writeShellApplication {
            name = "grepquest-init-sui-config";
            runtimeInputs = [
              pkgs.coreutils
              pkgs.gnused
            ];
            text = ''
              : "''${SUI_CONFIG_DIR:?SUI_CONFIG_DIR must be set}"
              umask 077
              install -d -m 0700 "$SUI_CONFIG_DIR"
              install -m 0600 ${suiDevKeystore} "$SUI_CONFIG_DIR/sui.keystore"

              temporary_config="$(mktemp "$SUI_CONFIG_DIR/.client.yaml.XXXXXX")"
              trap 'rm -f "$temporary_config"' EXIT
              sed "s|@SUI_CONFIG_DIR@|$SUI_CONFIG_DIR|g" ${suiDevClientConfigTemplate} > "$temporary_config"
              chmod 0600 "$temporary_config"
              mv "$temporary_config" "$SUI_CONFIG_DIR/client.yaml"
              trap - EXIT
            '';
          };
          devDoctor = pkgs.writeShellApplication {
            name = "grepquest-dev-doctor";
            runtimeInputs = [
              pkgs.go_1_25
              suiCli
            ]
            ++ pkgs.lib.optionals (system == "x86_64-linux") [ pkgs.firecracker ];
            text = ''
              failed=0

              check_command() {
                name="$1"
                if command -v "$name" >/dev/null 2>&1; then
                  printf '%-12s %s\n' "$name" "$(command -v "$name")"
                else
                  printf '%-12s %s\n' "$name" "missing"
                  failed=1
                fi
              }

              check_command go
              check_command sui
              check_command grepquest-ascii-motion
              check_command resvg

              if [ "$(uname -m)" = "x86_64" ]; then
                check_command firecracker
                check_command jailer
                if [ -c /dev/kvm ] && [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
                  printf '%-12s %s\n' "/dev/kvm" "readable and writable"
                else
                  printf '%-12s %s\n' "/dev/kvm" "unavailable or permission denied"
                  failed=1
                fi
              else
                printf '%-12s %s\n' "firecracker" "unsupported by this repository on $(uname -m)"
              fi

              printf '%-12s %s\n' "Sui version" "$(sui --version)"
              printf '%-12s %s\n' "Sui config" "$SUI_CONFIG_DIR (isolated, no keys)"
              printf '%-12s %s\n' "Firecracker" "$(firecracker --version 2>/dev/null || printf 'not available')"
              exit "$failed"
            '';
          };
          firecrackerAcceptance = pkgs.writeShellApplication {
            name = "grepquest-test-firecracker";
            runtimeInputs = [
              pkgs.coreutils
              pkgs.go_1_25
              pkgs.nix
            ];
            text = ''
              if [ "$(uname -m)" != "x86_64" ]; then
                echo "Firecracker acceptance testing is supported only on x86_64 Linux." >&2
                exit 1
              fi
              if [ ! -c /dev/kvm ] || [ ! -r /dev/kvm ] || [ ! -w /dev/kvm ]; then
                echo "/dev/kvm is unavailable or not readable and writable by the current user." >&2
                exit 1
              fi

              export GREPQUEST_FIRECRACKER="${pkgs.firecracker}/bin/firecracker"
              GREPQUEST_KERNEL="$(nix build --no-link --print-out-paths path:.#guestKernel)"
              GREPQUEST_INITRD="$(nix build --no-link --print-out-paths path:.#guestInitrd)/initrd"
              GREPQUEST_ROOTFS="$(nix build --no-link --print-out-paths path:.#guestRootfs --cores 2 --max-jobs 1)"
              export GREPQUEST_KERNEL GREPQUEST_INITRD GREPQUEST_ROOTFS
              export GREPQUEST_FIRECRACKER_INTEGRATION=1

              go test -count=1 -run '^TestIntegrationRealVM$' ./internal/sandbox/firecracker
            '';
          };
          firecrackerHostCheck = pkgs.writeShellApplication {
            name = "grepquest-firecracker-host-check";
            runtimeInputs = [
              pkgs.coreutils
              pkgs.firecracker
              pkgs.glibc.bin
              pkgs.gnugrep
              pkgs.gnused
            ];
            text = builtins.readFile ./tools/firecracker-host-check.sh;
          };
        in
        {
          default = pkgs.mkShell (
            {
              packages =
                with pkgs;
                [
                  bun
                  go_1_25
                  gopls
                  gotools
                  govulncheck
                  golangci-lint
                  just
                  nodejs_22
                  sqlite
                  gnugrep
                  nixfmt
                  resvg
                  suiCli
                  asciiMotionMcp
                  grepquestAsciiMotion
                  devDoctor
                ]
                ++ pkgs.lib.optionals (system == "x86_64-linux") [
                  firecracker
                  firecrackerAcceptance
                  firecrackerHostCheck
                ];
              SUI_CLI_VERSION = suiVersion;
              shellHook = ''
                export SUI_CONFIG_DIR="$PWD/.direnv/sui-config"
                ${suiDevConfigInit}/bin/grepquest-init-sui-config
              '';
            }
            // pkgs.lib.optionalAttrs (system == "x86_64-linux") {
              GREPQUEST_FIRECRACKER = "${pkgs.firecracker}/bin/firecracker";
            }
          );
        }
      );
    };
}
