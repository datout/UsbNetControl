# SignPath Foundation setup notes

This directory contains repository-side preparation for SignPath Foundation Open Source Code Signing.

## Before applying

Confirm all of the following:

- The GitHub repository is public and has at least one real release.
- The project is MIT licensed and all distributed source is open source.
- The README documents what the application does and where to download it.
- GitHub two-factor authentication is enabled for the maintainer account.
- `README.md` links to `CODE_SIGNING.md` and `PRIVACY.md`.
- The current GitHub Release description contains a `Code signing policy` link.

## Files in this preparation

- `../CODE_SIGNING.md`: project code signing policy and team roles.
- `../PRIVACY.md`: privacy policy.
- `artifact-configuration.xml`: SignPath artifact configuration template. It signs both Windows executables and enforces `ProductName=UsbNetControl` and the release `ProductVersion`.
- `release-after-approval.yml`: GitHub Actions workflow template to use only after approval.

## After SignPath Foundation approval

Do not replace the normal release workflow until the SignPath organization/project has been provisioned.

1. Install the SignPath GitHub App for `datout/UsbNetControl` when instructed by SignPath.
2. Configure the SignPath project to use GitHub.com as the Trusted Build System.
3. Create/import the artifact configuration from `artifact-configuration.xml`.
4. Add the following repository Actions configuration:

Secret:

- `SIGNPATH_API_TOKEN`

Variables:

- `SIGNPATH_ORGANIZATION_ID`
- `SIGNPATH_PROJECT_SLUG`
- `SIGNPATH_SIGNING_POLICY_SLUG`
- `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG`

5. Replace `.github/workflows/release.yml` with `signpath/release-after-approval.yml`.
6. Commit the workflow change before creating the next release tag.
7. Each release signing request must be manually approved in SignPath.

The SignPath workflow deliberately uploads the unsigned executables as a GitHub Actions artifact before submitting the signing request so SignPath can verify GitHub build origin.
