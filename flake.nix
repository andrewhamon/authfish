{
  description = "Packages and modules for Authfish";

  inputs.nixpkgs.url = "nixpkgs/nixos-22.11";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem
    (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in
      {
        packages = rec {
          authfish = import ./default.nix { inherit pkgs; };
          default = authfish;
        };

        devShells.default = import ./shell.nix { inherit pkgs; };
      }
    ) // {
    lib = import ./nix/lib.nix;
    nixosModules.default = { pkgs, ... }: {
      nixpkgs.overlays = [
        (self: super:
          {
            authfish = import ./default.nix { inherit pkgs; };
          })
      ];
      imports = [
        ./nix/modules/authfish.nix
      ];
    };
  };
}
