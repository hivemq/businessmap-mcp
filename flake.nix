{
  description = "BusinessMap MCP Server - A Model Context Protocol server for Kanbanize integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};

        version =
          if builtins.pathExists ./VERSION
          then builtins.readFile ./VERSION
          else if self ? shortRev
          then self.shortRev
          else "dev";

        businessmap-mcp = pkgs.buildGo124Module {
          pname = "businessmap-mcp";
          version = version;

          src = ./.;

          vendorHash = "sha256-+QOnYU+9OD4T+aEuHBOq6K/NCXuHPngFtDYl0KRSuSQ=";

          buildFlags = ["-mod=readonly"];

          ldflags = [
            "-X main.BuildVersion=${version}"
            "-s"
            "-w"
          ];

          meta = with pkgs.lib; {
            description = "Model Context Protocol server for Kanbanize integration";
            homepage = "https://github.com/hivemq/businessmap-mcp";
            license = licenses.asl20;
            maintainers = [];
            platforms = platforms.unix;
          };
        };
      in {
        packages = {
          default = businessmap-mcp;
          businessmap-mcp = businessmap-mcp;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-outline
            delve
          ];

          shellHook = ''
            echo "🔧 BusinessMap MCP development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available tools:"
            echo "  go build/run/test - Standard Go commands"
            echo "  gopls            - Go language server"
            echo "  dlv              - Go debugger (delve)"
            echo ""
            echo "Quick start:"
            echo "  go run . -version"
            echo "  go test ./..."
          '';
        };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
