# Authentication

BareRTC supports custom (user-defined) authentication with your app in the form of JSON Web Tokens (JWTs). JWTs will allow your existing app to handle authentication for users by signing a token that vouches for them, and the BareRTC app will trust your signed token.

The workflow is as follows:

1. Your existing app already has the user logged-in and you trust who they are. To get them into the chat room, your server signs a JWT token using a secret key that both it and BareRTC knows.
2. Your server redirects the user to your BareRTC website sending the JWT token as a `jwt` parameter, either in the query string (GET) or POST request.
    * e.g. you send them to `https://chat.example.com/?jwt=TOKEN`
    * If the JWT token is too long to fit in a query string, you may create a `<form>` with `method="POST"` that posts the `jwt` as a form field.
3. The BareRTC server will parse and validate the token using the shared Secret Key that only it and your back-end website knows.

There are JWT libraries available for most programming languages.

Configure a shared secret key (random text string) in both the BareRTC settings and in your app, and your app will sign a JWT including claims that look like the following (using signing method HS264):

```javascript
// JSON Web Token "claims" expected by BareRTC
{
    // Custom claims
    "sub": "username", // Username for chat (standard JWT claim)
    "op": true,        // User will have admin/operator permissions.
    "nick": "Display name",               // Friendly name
    "img": "/static/photos/username.jpg", // user picture URL
    "url": "/u/username",                 // user profile URL
    "gender": "m",                        // gender (m, f, o)
    "emoji": "🤖",                        // emoji icon
    "rules": ["redcam", "noimage"],       // moderation rules (optional)

    // Standard JWT claims that we support:
    "iss": "my own app", // Issuer name
    "exp": 1675645084,   // Expires at (time): 5 minutes out is plenty!
    "nbf": 1675644784,   // Not Before (time)
    "iat": 1675644784,   // Issued At (time)
}
```

**Notice:** your picture and profile URL may be relative URIs beginning with a forward slash as seen above; BareRTC will append them to the end of your WebsiteURL and you can save space on your JWT token size this way. Full URLs beginning with `https?://` will also be accepted and used as-is.

See [Custom JWT Claims](#custom-jwt-claims) for more information on the
custom claims and how they work.

An example how to sign your JWT tokens in Go (using [golang-jwt](https://github.com/golang-jwt/jwt)):

```golang
import "github.com/golang-jwt/jwt/v4"

// JWT signing key - keep it a secret on your back-end shared between
// your app and BareRTC, do not use it in front-end javascript code or
// where a user can find it.
const SECRET = "change me"

// Your custom JWT claims.
type CustomClaims struct {
    // Custom claims used by BareRTC.
    Avatar     string `json:"img"`  // URI to user profile picture
    ProfileURL string `json:"url"`  // URI to user's profile page
    IsAdmin    bool   `json:"op"`   // give operator permission

    // Standard JWT claims
    jwt.RegisteredClaims
}

// Assuming your internal User struct looks anything at all like:
type User struct {
    Username       string
    IsAdmin        bool
    ProfilePicture string  // like "/static/photos/username.jpg"
}

// Create a JWT token for this user.
func SignForUser(user User) string {
    claims := CustomClaims{
        // Custom claims
        ProfileURL: "/users/" + user.Username,
        Avatar:     user.ProfilePicture,
        IsAdmin:    user.IsAdmin,

        // Standard claims
        Subject:   user.Username, // their chat username!
        ExpiresAt: time.Now().Add(5 * time.Minute),
        IssuedAt:  time.Now(),
        NotBefore: time.Now(),
        Issuer:    "my own app",
        ID:        user.ID,
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, err := token.SignedString(SECRET)
    if err != nil {
        panic(err)
    }
    return tokenstr
}
```

## Custom JWT Claims

With JWT authentication your website can pass a lot of fun variables to decorate your Who Is Online list for your users.

Here is in-depth documentation on what custom claims are supported by BareRTC and what effects they have in chat:

* **Subject** (`sub`): this is a standard JWT claim and BareRTC will collect your username from it. The username is shown in the Who's Online list and below the user's nickname on their chat messages (in "@username" format). Do not prefix your subject with the @ symbol yourself.
* **Operator** (`op`): this boolean will mark your user to have operator (admin) status in chat. In the Who List they will have a gavel icon after their username, and they will be allowed to run operator commands (e.g. to kick other users from chat).
* **Nickname** (`nick`): you may send your users in with a custom Display Name that will appear on their chat messages. If they don't have a nickname, their username will be used in its place.
* **Image** (`img`): a profile picture or avatar for your users. It should be a square image and will appear in the Who List and alongside their chat messages. If they don't have an image, a default blue silhouette avatar is used. The image URL may be a relative URI beginning with `/` and it will be appended onto your configured WebsiteURL.
* **Profile URL** (`url`): a link to a user's profile page. If provided, clicking on their picture in chat or the Who List will open this URL in a new tab. They will also get a profile button added next to their name on the Who List. Relative URLs beginning with `/` are supported, and will be appended to your WebsiteURL automatically.
* **Gender** (`gender`): a single-character gender code for your user. If they also have a Profile URL, their profile button on the Who List can be color-coded by gender. Supported values include:
    * **m** (male) to set their profile button blue.
    * **f** (female) to set their profile button pink.
    * Other value (canonically, **o**) to set their profile button purple.
    * Missing/no value won't set a color and it will be the default text color.
* **Emoji** (`emoji`): you may associate users with an emoji character that will appear on the Who List next to their name. Some example ideas and use cases include:
    * Country flag emojis, to indicate where your users are connecting from.
    * Robot emojis, to indicate bot users.
    * Any emoji you want! Mark your special guests or VIP users, etc.
* **Rules** (`rules`): a string array of moderation rules to apply to the joining user, dictated by your website.
    * See [JWT Moderation Rules](./Configuration.md#jwt-moderation-rules) for available values.

## JWT Strict Mode

You can enable JWT authentication in a mixed mode: users presenting a valid token will get a profile picture and operator status (if applicable) and users who don't have a JWT token are asked to pick their own username and don't get any special flair.

In strict mode (default/recommended), only a valid JWT token can sign a user into the chat room. Set `[JWT]/Strict=false` in your settings.toml to disable strict JWT verification and allow "guest users" to log in. Note that this can have the same caveats as [running without authentication](#running-without-authentication) and is not a recommended use case.

## Running Without Authentication

The default app doesn't need any authentication at all: users are asked to pick their own username when joining the chat. The server may re-assign them a new name if they enter one that's already taken.

It is not recommended to run in this mode as admin controls to moderate the server are disabled.

### Known Bugs Running Without Authentication

This app is not designed to run without JWT authentication for users enabled. In the app's default state, users can pick their own username when they connect and the server will adjust their name to resolve duplicates. Direct message threads are based on the username so if a user logs off, somebody else could log in with the same username and "resume" direct message threads that others were involved in.

Note that they would not get past history of those DMs as this server only pushes _new_ messages to users after they connect.
