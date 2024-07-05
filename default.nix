{
  config,
  lib,
  dream2nix,
  ...
}: let
  pyproject = lib.importTOML (config.mkDerivation.src + /pyproject.toml);
in {
  buildPythonPackage = {
    pyproject = true;
    pythonImportsCheck = [
      "nba-chatbot"
    ];
  };
  deps = {nixpkgs, ...}: {
    python = nixpkgs.python3;
  };
  imports = [
    dream2nix.modules.dream2nix.pip
  ];
  inherit (pyproject.project) name version;
  mkDerivation = {
    src = lib.cleanSourceWith {
      filter = name:
        !(builtins.any (x: x) [
          (lib.hasSuffix ".nix" name)
          (lib.hasPrefix "." (builtins.baseNameOf name))
          (lib.hasSuffix "flake.lock" name)
        ]);
      src = lib.cleanSource ./.;
    };
  };
  pip = {
    editables.charset-normalizer = ".editables/charset_normalizer";
    flattenDependencies = true;
    requirementsList =
      pyproject.build-system.requires
      or []
      ++ pyproject.project.dependencies;
    overrides.click = {
      buildPythonPackage.pyproject = true;
      mkDerivation.nativeBuildInputs = [config.deps.python.pkgs.flit-core];
    };
  };
}
