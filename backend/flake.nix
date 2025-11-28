{
  description = "MCP Server in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        mcp-server = pkgs.buildGoModule rec {
          pname = "mcp-server";
          version = "0.1.0";
          src = ./.;
          vendorHash = null; # Will be updated after go mod vendor

          nativeBuildInputs = with pkgs; [
            pkg-config
          ];

          buildInputs = with pkgs; [
            sqlite
          ];

          ldflags = [
            "-s"
            "-w"
          ];
        };
      in
      {
        packages = {
          default = mcp-server;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            go-tools
            golangci-lint
            sqlite
            sqlite-interactive
            pkg-config
            air # for hot reloading during development
          ];

          shellHook = ''
            echo "MCP Server Development Environment"
            echo "=================================="
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go run .                  - Run the MCP server"
            echo "  go build                  - Build the MCP server"
            echo "  go test ./...             - Run tests"
            echo "  air                       - Run with hot reload"
            echo "  sqlite3 database.db       - Open SQLite database"
            echo ""
            echo "Environment variables:"
            echo "  OPENROUTER_API_KEY        - Set your OpenRouter API key"
            echo "  DATABASE_PATH             - Path to SQLite database (default: ./database.db)"
            echo ""
            
            # Create database if it doesn't exist
            if [ ! -f database.db ]; then
              echo "Creating initial database..."
              sqlite3 database.db < schema.sql
              echo "Database created at database.db"
            fi
          '';

          OPENROUTER_API_KEY = builtins.getEnv "OPENROUTER_API_KEY";
        };
      });
}