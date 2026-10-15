# Security Policy

## Supported versions

The latest tagged release receives security fixes.

## Reporting a vulnerability

Please report suspected vulnerabilities privately rather than opening a public
issue. Use GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
for this repository.

Include a description, a minimal reproduction, and the affected version. We aim
to acknowledge reports within a few business days.

## Scope

This library encodes data to QR symbols; it does not decode untrusted input and
does not perform I/O beyond writing to a caller-provided `io.Writer`. The most
likely classes of issue are:

- Panics or excessive resource use on adversarial *encode* input (covered by
  fuzz tests).
- Incorrect bounds handling in the renderers.

Decoding of untrusted symbols is out of scope, as the library does not decode.
