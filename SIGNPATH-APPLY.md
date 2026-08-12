# SignPath Foundation application checklist

Repository: `https://github.com/datout/UsbNetControl`

Before submitting the application:

1. Push the SignPath preparation commit to `main`.
2. Verify GitHub 2FA is enabled for `datout`.
3. Ensure the public `v1.3.1` Release remains available.
4. Add the Code signing policy link to the existing `v1.3.1` Release description.
5. Apply at `https://signpath.org/apply` and provide the public repository/release details requested by the form.

The project is a local Windows administrative utility. It does not contain vulnerability scanning/exploitation features, telemetry or background networking. Its system changes are explicit user-requested actions: removable-storage access policy changes and network-adapter enable/disable operations.

Do not enable `signpath/release-after-approval.yml` before SignPath provides the organization/project identifiers and API token.
