{
  description =
    "A nix-flake-based Go/Kubernetes Controller development environment using operator-sdk";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      regex = "go[[:space:]]+([0-9]+\\.[0-9]+)(\\.[0-9]+)?";
      # Read and get all the lines of the go.mod files and filter the empty []
      lines = builtins.filter (line: line != [ ])
        (builtins.split "\n" (builtins.readFile ./go.mod));
      # Get the line that has the `go xx.xx.xx` or `go xx.xx`
      matchingLines =
        builtins.filter (line: builtins.match regex line != null) lines;
      # Get the version string i.e: "1.23"
      versionString = builtins.elemAt
        (builtins.match regex (builtins.concatStringsSep " " matchingLines)) 0;
      # Get the string as "1_23"
      versionForOverlay = builtins.replaceStrings [ "." ] [ "_" ] versionString;

      supportedSystems =
        [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f:
        nixpkgs.lib.genAttrs supportedSystems (system:
          f {
            pkgs = import nixpkgs {
              inherit system;
              overlays = [ self.overlays.default ];
            };
          });
    in {
      overlays.default = final: prev: { go = final."go_${versionForOverlay}"; };

      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # go (version is specified by overlay)
            go

            # gopls (go language server)
            gopls

            # goimports, godoc, etc.
            gotools

            # SDK for building Kubernetes applications
            operator-sdk
          ];
        };
      });
    };
}
