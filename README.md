# Authfish
Authfish is a simple identity provider intended to be used in conjunction with
nginx's `auth_request` feature. Authfish specifically targets small, self-hosted
use cases. It does not rely on any external services to function.

## Usage with nixos

Authfish is packaged as a nix flake, and also provides a NixOS module for running it as a service. There is also a small utility library for easily protecting a NixOS nginx virtual host with Authfish.

These examples assume you are passing your flake inputs to your modules using specialArgs as `inputs`. See [this informative blog post](https://blog.nobbz.dev/posts/2022-12-12-getting-inputs-to-modules-in-a-flake/#extraspecialargs) for an examples of how to do that.

### Configuring the Authfish service

Example nixos configuration:
```nix
{config, inputs, ...}:
{
  imports = [
    inputs.authfish.nixosModules.default
  ];

  services.authfish.enable = true;

  # List of domains you want authfish to protect.
  # Use `.example.com` instead of example.com
  # to use one cookie for all subdomains of example.com.
  services.authfish.domains = [".example.com"];

  # Domain where you want to host the authfish UI.
  # This is where registration links are handled.
  services.authfish.virtualHostName = "login.example.com";

  # These are passed through to the underlying nginx config for
  # "login.example.com". These should be true in production.
  services.authfish.enableACME = true;
  services.authfish.forceSSL = true;
}
```


### Protecting resources with authfish

Protecting an nginx virtual host is easy, just wrap your existing nginx config
with `protectWithAuthfish`. The first argument to `protectWithAuthfish` is
`config`, which the lib needs to determine which port authfish is listening on.

```nix
{ config, inputs, ... }:
let
  # Note that we are currying `config`.
  protectWithAuthfish = inputs.authfish.lib.protectWithAuthfish config;
in
{
  services.nginx.virtualHosts."app.example.com" = protectWithAuthfish {
    enableACME = true;
    forceSSL = true;
    locations."/" = {
      proxyPass = "http://localhost:1234";
    };
  };
}
```

### Managing users

**Add user**
```sh
sudo -u authfish authfish user add bob
```

Which will output:
```
/register?registrationToken=<token>
```

**List users**

```sh
sudo -u authfish authfish user list
```

Which will output:

```
Id	Username	Registration URL                                            	Created At                   	Updated At
1 	bob     	/register?registrationToken=<token>	2023-01-09 02:03:50 +0000 UTC	2023-01-09 02:03:50 +0000 UTC
```

**Delete user**

```sh
sudo -u authfish authfish user remove bob
```

Which will output:

```
Deleted user bob
```
