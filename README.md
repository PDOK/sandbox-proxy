# Sandbox Proxy

This Sandbox Proxy is used to setup a local tunnel to the PDOK sandbox environment. This proxy 
handles both routing and security.

## HowTo

1. Create a Public/Private key pair. For linux use the following command:

    ```bash
    $ openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048
    ```

    or for Windows use the following tutorial: https://help.singlecomm.com/hc/en-us/articles/115008214927-Generating-Public-Private-RSA-Keys

2. Contact PDOK to request a sandbox environment. Include the public key (**not private key**) in your request. After 
   the sandbox environment is created PDOK wil provide you with the name of the sandbox environment.
   
3. Start the `sandbox-proxy` application on your local machine. Below are examples for Docker, Linux and Windows.

## Usage

### Docker

```bash
docker run -v ${PWD}:/tmp -p 5000:5000 -p 5001:5001 -p 5002:5002 -p 5003:5003 -p 5004:5004 -p 5005:5005 \
  -p 5006:5006 pdok/sandbox-proxy:latest --sandbox-name <name> --private-key /tmp/<private key file>
```

### Linux

```bash
sandbox-proxy --sandbox-name <name> --private-key <private key file>
```

### Windows

```bat
sandbox-proxy --sandbox-name <name> --private-key <private key file>
```
