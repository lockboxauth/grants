# grants

The `grants` package encapsulates the part of the [auth system](https://impractical.co/auth) that handles authentication requests and turns them into sessions.

Put another way, it handles the "login" requests from users and calls through to the appropriate other pieces of the auth system to log the user in.

## Implementation

Grants consist of an ID, a source type, a source ID, the scopes they cover, the profile they're for, and some metadata. The source type and source ID are used to avoid reuse attacks; the combination of source ID and source type must be unique amongst all grants. The source ID is meaningful only within the context of that specific grant type; they are opaque within the system as a whole.

## Place in the Architecture

When a user first logs in, they create a grant request. For example, when authenticating with an email address, a grant is created with the grant type "email" and a uniquely generated grant ID. The user then clicks the link in their email, which contains the grant ID, which the grants service uses to create a session. For the Google OpenID login flow, there's only one step: pass an ID token from Google, so the grant type is "google_id", and the grant ID is an encoding of some uniquely identifying information from the ID token.

Once a grant is obtained, it can be exchanged for a session.

## Scope

`grants` is solely responsible for managing the various login methods, verifying and validating them, and converting them into a unified representation in the system. The HTTP handlers it provides are responsible for verifying the authentication and authorization of the requests made against it, which will be coming from untrusted sources.

The questions `grants` is meant to answer for the system include:

  * Is this a valid authentication request?
  * What scopes should the session have?
  * How did a session get created?

The things `grants` is explicitly not expected to do include:

  * Manage valid scopes.
  * Manage the mapping of login methods to profile IDs.
  * Serve as anything more than an audit log of authentications.
