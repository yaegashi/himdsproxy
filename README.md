# himdsproxy

himdsproxy is a simple proxy for the Hybrid Instance Metadata Service (HIMDS) on [Azure Arc-enabled servers](https://learn.microsoft.com/en-us/azure/azure-arc/servers/overview).

himdsproxy translates requests for the IMDS endpoint `http://169.254.169.254` (served by [the Azure VM infrastructure](https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service)) to requests for the HIMDS endpoint `http://localhost:40342` (served by [the Azure Connected Machine agent](https://learn.microsoft.com/en-us/azure/azure-arc/servers/agent-overview)).

himdsproxy allows you to obtain various instance metadata and OAuth2 access tokens ([managed identities for Azure resources](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/overview)) on on-premises servers, as if they were from the non-Hybrid IMDS, which is only available for Azure VMs.

One of the main use cases of himdsproxy is to enable [the non-Hybrid Entra Join](https://learn.microsoft.com/en-us/entra/identity/devices/concept-directory-join) on non-Azure VM Windows Servers using [RunAADLoginForWindows.ps1](RunAADLoginForWindows.ps1).  It utilizes the handler of [the AADLoginForWindows Azure VM extension](https://learn.microsoft.com/en-us/entra/identity/devices/howto-vm-sign-in-azure-ad-windows), which is not originally compatible with Azure Arc-enabled Windows Servers.  This helps you to eliminate the AD DS requirements by using cloud-native user authentication with the non-Hybrid Entra Joined Windows Servers.
