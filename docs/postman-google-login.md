# Logging in with Google from Postman

How to get a session token for the GraphQL API from Postman, without the Web Console.

The flow has two parts:

1. Postman signs in to Google and gets a Google **ID token**.
2. You send that ID token to the `loginWithGoogle` mutation, which returns this service's own **session token**.
   All other requests use the session token.

You only need Google again when the session token expires (`JWT_TTL`, default `24h`).

## 1. One-time setup in Google Cloud

In [Google Cloud Console](https://console.cloud.google.com/) → **Google Auth Platform** → **Clients**, open the
OAuth client whose ID is in your `GOOGLE_CLIENT_ID`:

- Under **Authorized redirect URIs**, add `https://oauth.pstmn.io/v1/callback`.
  This is Postman's callback page. It goes under *redirect URIs*, **not** *Authorized JavaScript origins*,
  which only apply to the browser sign-in button.
- Copy the client's **Client ID** and **Client secret**. Postman needs the secret for this flow; the service
  itself never does.

If the app's publishing status is **Testing**, add your Google account under **Audience → Test users**.
Otherwise Google refuses the sign-in.

## 2. One-time setup in Postman

Create an **environment** (e.g. `finance-core local`) with these variables:

| Variable | Value |
|---|---|
| `graphqlUrl` | `http://localhost:8080/graphql` (local mode) or `http://localhost:8180/graphql` (Docker mode) |
| `googleClientId` | Your OAuth client ID |
| `googleClientSecret` | Your OAuth client secret |
| `googleIdToken` | Leave empty; filled in step 3 |
| `sessionToken` | Leave empty; filled in step 4 |

Put the client secret and both tokens in the **Current value** column only, and leave **Initial value**
empty. Postman syncs initial values to its cloud and includes them when sharing or exporting; current values
stay on your machine. Never commit the secret or a token anywhere.

## 3. Get a Google ID token

Create a request (any URL; it's only used for its Authorization tab), open **Authorization**, and set
**Type** to **OAuth 2.0**. Under **Configure New Token**:

| Field | Value |
|---|---|
| Grant type | `Authorization Code (With PKCE)` |
| Callback URL | Tick **Authorize using browser** (fills in `https://oauth.pstmn.io/v1/callback`) |
| Auth URL | `https://accounts.google.com/o/oauth2/v2/auth` |
| Access Token URL | `https://oauth2.googleapis.com/token` |
| Client ID | `{{googleClientId}}` |
| Client Secret | `{{googleClientSecret}}` |
| Code Challenge Method | `SHA-256` |
| Scope | `openid email profile` |
| Client Authentication | `Send client credentials in body` |

Click **Get New Access Token**, sign in with Google in the browser window, and return to Postman.

Postman shows the token response. It contains both an `access_token` and an **`id_token`**. Copy the
**`id_token`** (a long string starting with `eyJ`) into the `googleIdToken` environment variable.

Use the `id_token`, not the `access_token`: the service verifies the ID token's signature and audience, and
an access token fails with `UNAUTHENTICATED`.

The Google ID token is valid for about **1 hour**. If `loginWithGoogle` returns `UNAUTHENTICATED` later,
get a new one.

Field labels can differ slightly between Postman versions; the values stay the same.

## 4. Exchange it for a session token

Create a **POST** request to `{{graphqlUrl}}`, with body type **GraphQL**:

```graphql
mutation {
  loginWithGoogle(idToken: "{{googleIdToken}}") {
    token
    expiresAt
    user { id email firstName lastName telegramChatId }
  }
}
```

The first login creates your user from your Google profile; later logins return the same user.

To save the session token automatically, add this under the request's **Scripts → Post-response**:

```js
const data = pm.response.json().data;
if (data && data.loginWithGoogle) {
  pm.environment.set("sessionToken", data.loginWithGoogle.token);
}
```

## 5. Use the session token

For every other request, set **Authorization → Type** to **Bearer Token** with token `{{sessionToken}}`.
Postman then sends `Authorization: Bearer <session token>`.

To set it once for all requests, put it on the **collection**'s Authorization tab, and set each request's
Authorization to **Inherit auth from parent**. Set the `loginWithGoogle` request itself to **No Auth**.

Check it works:

```graphql
query {
  me { id email firstName lastName telegramChatId }
}
```

When the session token expires, repeat steps 3 and 4.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Google shows `redirect_uri_mismatch` | The Postman callback URL isn't registered | Add `https://oauth.pstmn.io/v1/callback` under **Authorized redirect URIs** (step 1) |
| Google shows `access_denied` / "app is being tested" | Publishing status is Testing and your account isn't a test user | Add your account under **Test users**, or publish the app |
| `loginWithGoogle` returns `UNAUTHENTICATED` | Expired ID token (over ~1 hour old); an `access_token` sent instead of the `id_token`; or the token was issued for a different client ID than the server's `GOOGLE_CLIENT_ID` | Get a new ID token; check you copied `id_token`; check both use the same OAuth client |
| `loginWithGoogle` returns `UNAUTHENTICATED` for a valid token | The Google account's email isn't verified | Use an account with a verified email |
| `loginWithGoogle` returns `CONFLICT` | Another Google account already registered with the same email | Sign in with the Google account that registered first |
| `me` returns `UNAUTHENTICATED` | Missing, expired or modified session token | Check the request's Authorization is Bearer `{{sessionToken}}`; repeat steps 3–4 |
