# Security Policy

The Heimdall team takes the security of our project seriously. We appreciate the efforts of security researchers and the community to help us maintain a high standard of security.

If you believe you have found a security vulnerability in Heimdall or any of its components, we strongly encourage you to report it to us privately.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please use one of the following private methods:

### 1. GitHub Private Vulnerability Reporting (Preferred)

We strongly prefer that you use GitHub's **Private Vulnerability Reporting** feature. This is the most secure and efficient way to report a vulnerability and collaborate on a fix.

You can submit a private report by:

1. Navigating to the main page of the Heimdall repository.
    
2. Under the repository name, click the **Security** tab.
    
3. Click **Report a vulnerability**.
    
4. Fill out the form with the details of the vulnerability.
    

This method ensures the report goes directly to the project maintainers and is not publicly visible.

### 2. Email Reporting

If you are unable to use GitHub's private reporting, you can send an email to: **`security-heimdall@nexlified.com`**

Please include a detailed description of the vulnerability, steps to reproduce it, and any potential impact.

## Our Commitment

When you report a vulnerability through these private channels, you can expect us to :

- Respond to your report in a timely manner, typically within 48 hours.
    
- Work with you to understand and validate your report.
    
- Keep you informed of our progress as we work to remediate the issue.
    
- Strive to fix the vulnerability in a timely manner.
    
- Publicly credit you for your contribution (if you wish) once the vulnerability has been addressed.
    

## Scope

We are interested in vulnerabilities in:

- The Heimdall-core services
    
- Our official Docker images and `docker-compose.yml` configuration
    
- The `heimdall-example-app`
    

The following are considered **out-of-scope** :

- Denial of Service (DoS) attacks on any of our infrastructure.
    
- Social engineering or phishing attacks against the maintainers or community.
    
- Reports on outdated dependencies that have not yet been patched (unless you can demonstrate a specific exploit).
    
- Vulnerabilities in the underlying open-source components (e.g., Kratos, Cerbos, NATS) unless they are caused by a specific misconfiguration in Heimdall. Please report those to the respective projects.
    

## Safe Harbor

We consider security research conducted under this policy to be authorized and will not pursue or support any legal action related to your research. We ask that you make a good faith effort to avoid privacy violations, data destruction, and interruption of our services.
