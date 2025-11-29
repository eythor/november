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
            # sqlite
            sqlite-interactive
            pkg-config
            air # for hot reloading during development
            jq # for JSON processing in utilities
            curl # for HTTP requests
          ];

          shellHook = ''
            echo "MCP Server Development Environment"
            echo "=================================="
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  make http-server          - Start HTTP server (recommended)"
            echo "  ./mcp-query-http          - Query the HTTP server"
            echo "  ./mcp-query               - Interactive query (stdin/stdout mode)"
            echo "  go run cmd/http/main.go   - Run HTTP server directly"
            echo "  go run .                  - Run the MCP server (stdin/stdout)"
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

          OPENROUTER_API_KEY = pkgs.builtins.getEnv "OPENROUTER_API_KEY";
        };
      });
}
