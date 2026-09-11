# Security policy

Security fixes target the latest 1.0.x release. Use current patches and rebuild containers when Go or dependency advisories require it.

Report suspected vulnerabilities privately to smmartbiz@gmail.com with reproduction steps and affected versions. Do not include service-account keys, bearer tokens or private Search Console data. Please allow investigation before public disclosure.

The bearer token grants access to all properties visible to the installation's service account. Use HTTPS for remote connections and a separate installation for each trust boundary. `/health` is intentionally unauthenticated and contains no Google data. Insecure mode is for loopback debugging only. Demo data is synthetic; demo mode still requires authentication by default.

Keep credentials outside version control and container images, restrict file permissions and rotate exposed secrets. Automated tests use fake Google responses and require no real credentials.
