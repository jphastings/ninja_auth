# Ninja Auth

This is a simple reverse proxy which requires authenitcation via Google OAuth and a matching _hosted_domain_ assertion from Google before allowing access to the locally hosted HTTP server behind.

The idea is to put this on Heroku slugs to provide an authentication layer to applications where building SSO in that language is prohibitively difficult.

## Set up

Get your dependencies and build:

```shell
go get ./...
go build ninja.go
```

Go create some [Google OAuth credentials](https://console.developers.google.com/apis/credentials). Ensure your chosen credentials have an authorized redirect that is your indended host for Ninja Auth, with the path `/ninja_auth`.

Ensure you have environment variables set:

Env Var                    | Description
---------------------------|------------
`PORT`                     | The port Ninja Auth should listen on
`NINJA_BASE_URL`           | The base URL for where Ninja Auth will be hosted, with _no path component_. If you want to test locally using [puma-dev](https://github.com/puma/puma-dev) this would be something like: `https://ninja.dev`
`NINJA_SECRET`             | The secret used for your cookies. Make it long and unguessable!
`NINJA_PROXY_PORT`         | The port on the loopback device that Ninja Auth will redirect requests to for authenticated users
`NINJA_ACCEPTABLE_DOMAINS` | A comma separated list of hosted domains which should be authorized
`GOOGLE_CLIENT_ID`         | The Client ID that Google gave you
`GOOGLE_CLIENT_SECRET`     | The Client Secret that Google gave you

## Things I want to improve

* Better unauthorized page (explaining why you can't view)
* Tests!
* Make a heroku buildpack to do set this all up on Heroku
