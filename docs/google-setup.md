# Google Search Console access

The MCP server authenticates to Google as a **service account**. That account must be added as a user on each Search Console property.

## 1. Google Cloud

1. Create or reuse a Google Cloud project.
2. Enable **Google Search Console API**.
3. Create a service account (no Google Cloud roles are required for Search Console itself).
4. Create a JSON key and store it as `credentials/service-account.json` on the server. Do not commit this file.

## 2. Search Console

1. Open [Search Console](https://search.google.com/search-console).
2. Open the property (URL-prefix `https://example.com/` or domain `sc-domain:example.com`).
3. Settings → Users and permissions → Add user.
4. Invite the service account email (`...@....iam.gserviceaccount.com`).
5. Full user is enough for this MVP (read-only tools).

The `siteUrl` you pass to tools must match the property string **exactly**, including the trailing slash on URL-prefix properties.

## 3. Verify

After deploy, ask the agent to call `list_sites`. You should see the properties you shared. If the list is empty, the service account is not added to those properties.
