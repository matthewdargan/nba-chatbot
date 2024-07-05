{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    parts.url = "github:hercules-ci/flake-parts";
    pre-commit-hooks = {
      inputs.nixpkgs.follows = "nixpkgs";
      url = "github:cachix/pre-commit-hooks.nix";
    };
  };
  outputs = inputs:
    inputs.parts.lib.mkFlake {inherit inputs;} {
      imports = [inputs.pre-commit-hooks.flakeModule];
      perSystem = {
        config,
        pkgs,
        ...
      }: let
        python = pkgs.python3.withPackages (ps: with ps; [langchain langchain-community]);
      in {
        devShells.default = pkgs.mkShell {
          packages = [pkgs.ollama python];
          shellHook = "${config.pre-commit.installationScript}";
        };
        pre-commit = {
          settings = {
            hooks = {
              alejandra.enable = true;
              deadnix.enable = true;
              ruff.enable = true;
              statix.enable = true;
            };
            src = ./.;
          };
        };
      };
      systems = ["aarch64-darwin" "aarch64-linux" "x86_64-darwin" "x86_64-linux"];
    };
}
