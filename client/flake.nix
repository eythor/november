{
  description = "Black forest Hackathon Doctor AI Assistant Development Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        pythonEnv = pkgs.python311.withPackages (ps: with ps; [
          pip
          ipython

          openai-whisper
        ]);
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            pythonEnv
            openai-whisper

            podman
            podman-compose
            sqlite
            curl
            wget
            jq
            git
          ];
          shellHook = ''
            echo "Black forest Hackathon Doctor AI Assistant Development Environment"
          '';
        };
      });
}
