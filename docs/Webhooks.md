# Webhook URLs

BareRTC supports setting up webhook URLs so the chat server can call out to _your_ website in response to certain events, such as allowing users to send you reports about messages they receive on chat.

Webhooks are configured in your settings.toml file and look like so:

```toml
[[WebhookURLs]]
  Name = "report"
  Enabled = true
  URL = "http://localhost:8080/v1/barertc/report"

[[WebhookURLs]]
  Name = "profile"
  Enabled = true
  URL = "http://localhost:8080/v1/barertc/profile"
```

All Webhooks will be called as **POST** requests and will contain a JSON payload that will always have the following two keys:

* `Action` will be the name of the webhook (e.g. "report")
* `APIKey` will be your AdminAPIKey as configure in the settings.toml (shared secret so your web app can authenticate BareRTC's webhooks).

The JSON payload may also contain a relevant object per the Action -- see the specific examples below.

## Report Webhook

Enabling this webhook will cause BareRTC to display a red "Report" flag button underneath user messages on chat so that they can report problematic messages to your website.

The webhook name for your settings.toml is "report"

Example JSON payload posted to the webhook:

```javascript
{
    "Action": "report",
    "APIKey": "shared secret from settings.toml#AdminAPIKey",
    "Report": {
        "FromUsername": "sender",
        "AboutUsername": "user being reported on",
        "Channel": "lobby",  // or "@username" for DM threads
        "Timestamp": "(stringified timestamp of chat message)",
        "Reason": "It's spam",
        "Comment": "custom user note about the report",
        "Message": "the actual message that was being reported on",
    }
}
```

BareRTC expects your webhook URL to return a 200 OK status code or it will surface an error in chat to the reporter.

## Profile Webhook

Enabling this webhook will allow your site to deliver more detailed profile information on demand for your users. This is used in chat when somebody opens the Profile Card modal for a user in chat.

BareRTC will call your webhook URL with the following payload:

```javascript
{
    "Action": "profile",
    "APIKey": "shared secret from settings.toml#AdminAPIKey",
    "Username": "soandso"
}
```

The expected response from your endpoint should follow this format:

```javascript
{
    "StatusCode": 200,
    "Data": {
        "OK": true,
        "Error": "any error messaging (omittable if no errors)",
        "Headline": "Optional profile headline text",
        "ProfileFields": [
            {
                "Name": "Age",
                "Value": "30yo",
            },
            {
                "Name": "Gender",
                "Value": "Man",
            },
            // etc.
        ]
    }
}
```

If the Headline field is provided from your site, it will be shown above the profile fields in a banner.