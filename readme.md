# Z-Notes

Z-Notes is a simple web-based note taking/saving service. This is forked and rebased from my go-image-board. It uses OpenID Connect for it's authentication back-end. I recommend using something like KeyCloak to provide the user accounts. Notes are markdown based and allow file uploads. This project is still in early development. 

## Installation

### Simple Docker Build

These steps will get you up and running immediately

1. Copy the executable, http/*, and the dockerfile to your build directory
2. cd to your build directory
3. Build the image
```
docker build -t z-notes .
```
4. Run a new instance of the z-notes
```
docker run --name z-notes -p 80:8080 -v /var/docker/z-notes/files:/var/z-notes/files -v /var/docker/z-notes/configuration:/var/z-notes/configuration -d z-notes
```
5. Stop the instance and edit the configuration file as needed
6. Start instance again

### Custom Docker Build

Similiar to the previous steps, the main difference here is that you are supplying your own template files to customize the look of the notes service.

1. Copy the executable, http/*, and the dockerfile to your build directory
2. cd to your build directory
3. Build the image
```
docker build -t z-notes .
```
4. Create a custom dockerfile that uses z-notes as it's parent, and add your necessary changes
```
FROM z-notes
COPY myhttp "/var/z-notes/http"
WORKDIR /var/z-notes
ENTRYPOINT ["/var/z-notes/z-notes"]
```
5. Run a new instance of z-notes
```
docker run --name z-notes -p 80:8080 -v /var/docker/z-notes/files:/var/z-notes/files -v /var/docker/z-notes/configuration:/var/z-notes/configuration -d z-notes
```
6. Stop the instance and edit the configuration file as needed
7. Start instance again

### TLS

Z-Notes supports TLS. I recommend using an Nginx reverse proxy container for its additional features and [letsencrypt](https://letsencrypt.org/) support. To enable TLS set UseTLS, TLSCertPath, and TLSKeyPath in your configuration file. Such as `{...,"UseTLS":true,"TLSCertPath":".\\configuration\\cert.pem","TLSKeyPath":".\\configuration\\server.key"}`

### Configuration File

When you run Z-Notes for the first time, the application will generate a new config file for you. This config file is JSON formatted and contains various configuration options. This file must be configured in order for Z-Notes to be usable. 

| Configuration Setting | Default | Use |
| --------------- | --------------- | --------------- |
| DBName | no default | The name of your database. Required, application will not function fully and show a message stating configuration required |
| DBUser | no default | The username to use when authenticating to the database. Required, application will not function fully and show a message stating configuration required |
| DBPassword | no default | The password to use when authenticating to the database. Required, application will not function fully and show a message stating configuration required |
| DBPort | no default | The port to use when authenticating to the database. Required, application will not function fully and show a message stating configuration required |
| DBHost | no default | The database host. Required, application will not function fully and show a message stating configuration required |
| PageDirectory | ./pages | The directory to save files embedded in notes to |
| Address | :8080 | Address for the server to listen on. Defaults to any ip over port 8080 |
| ReadTimeout | 30 seconds | Timeout for read operations on http server |
| WriteTimeout | 30 seconds | Timeout for write operations on http server |
| MaxHeaderBytes | 1MB | Maximum bytes allowed for http headers |
| SessionStoreKey | * | Key to the gorilla cookies session store. This will be automatically generated on first run |
| CSRFKey | * | Key to the gorilla CSRF token. This will be automatically generated on first run |
| HTTPRoot | ./http | Path to the golang http template files |
| MaxUploadBytes | 100MB | Maximum allowed size of uploads |
| AllowAccountCreation | false | If true, allows new users to register/sign-in |
| APIThrottle | 0 | Time required to wait between API calls. Currently unused |
| UseTLS | false | If true, server will use https. A certificate and key are required |
| TLSCertPath | no default | If UseTLS is set, this is the cert that will be used for https |
| TLSKeyPath | no default | If UseTLS is set, this should be the matching private key file for the tls cert |
| OpenIDClientID | no default | Your OpenID Connect Client ID, required |
| OpenIDClientSecret | no default | Your OpenID client secret, requirement dependent on OpenID provider setup |
| OpenIDCallbackURL | no default | The full URL to the callback endpoint. Required. This is dependent on your external domain name for the server. If running locally this would be "http://localhost:8080/openidc/callback". If running on a dedicated, externally accessible server, then this should look like "https://{server.domain.tld}/openidc/callback" |
| OpenIDEndpointURL | no default | The full URL to your OpenID Connect provider. Required. |
| OpenIDLogonExpireTime | 1209600 | The time to automatically expire OpenID tokens in seconds. |
| TargetLogLevel | 0 | Sets the verbosity of the log. |
| MaxQueryResults | 20 | Maximum results to return when querying notes |

## About files

Files located in the "/http/about/" directory are imported into the about.html template and served when requested from http://\<yourserver\>/about/\<filename\>.html
This can be used to easily write rules, or other documentation while maintaining the same general theme.
