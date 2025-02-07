// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"fmt"

	"github.com/Azure/aks-engine/pkg/api"
	"github.com/Azure/aks-engine/pkg/helpers"
)

func getParameters(cs *api.ContainerService, generatorCode string, aksEngineVersion string) paramsMap {
	properties := cs.Properties
	location := cs.Location
	parametersMap := paramsMap{}
	cloudSpecConfig := cs.GetCloudSpecConfig()

	// aksengine Parameters
	addValue(parametersMap, "aksEngineVersion", aksEngineVersion)

	// Master Parameters
	addValue(parametersMap, "location", location)

	// Identify Master distro
	if properties.MasterProfile != nil {
		addValue(parametersMap, "osImageOffer", cloudSpecConfig.OSImageConfig[properties.MasterProfile.Distro].ImageOffer)
		addValue(parametersMap, "osImageSKU", cloudSpecConfig.OSImageConfig[properties.MasterProfile.Distro].ImageSku)
		addValue(parametersMap, "osImagePublisher", cloudSpecConfig.OSImageConfig[properties.MasterProfile.Distro].ImagePublisher)
		addValue(parametersMap, "osImageVersion", cloudSpecConfig.OSImageConfig[properties.MasterProfile.Distro].ImageVersion)
		if properties.MasterProfile.ImageRef != nil {
			addValue(parametersMap, "osImageName", properties.MasterProfile.ImageRef.Name)
			addValue(parametersMap, "osImageResourceGroup", properties.MasterProfile.ImageRef.ResourceGroup)
		}
	}

	addValue(parametersMap, "fqdnEndpointSuffix", cloudSpecConfig.EndpointConfig.ResourceManagerVMDNSSuffix)
	addValue(parametersMap, "targetEnvironment", helpers.GetTargetEnv(cs.Location, cs.Properties.GetCustomCloudName()))
	linuxProfile := properties.LinuxProfile
	if linuxProfile != nil {
		addValue(parametersMap, "linuxAdminUsername", linuxProfile.AdminUsername)
		if linuxProfile.CustomNodesDNS != nil {
			addValue(parametersMap, "dnsServer", linuxProfile.CustomNodesDNS.DNSServer)
		}
	}
	if properties.MasterProfile != nil {
		// masterEndpointDNSNamePrefix is the basis for storage account creation for k8s
		addValue(parametersMap, "masterEndpointDNSNamePrefix", properties.MasterProfile.DNSPrefix)
		if properties.MasterProfile.IsCustomVNET() {
			addValue(parametersMap, "masterVnetSubnetID", properties.MasterProfile.VnetSubnetID)
			if properties.MasterProfile.IsVirtualMachineScaleSets() {
				addValue(parametersMap, "agentVnetSubnetID", properties.MasterProfile.AgentVnetSubnetID)
			}
			if properties.MasterProfile.VnetCidr != "" {
				addValue(parametersMap, "vnetCidr", properties.MasterProfile.VnetCidr)
			}
		} else {
			addValue(parametersMap, "masterSubnet", properties.MasterProfile.Subnet)
			addValue(parametersMap, "agentSubnet", properties.MasterProfile.AgentSubnet)
			if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") {
				addValue(parametersMap, "masterSubnetIPv6", properties.MasterProfile.SubnetIPv6)
			}
		}
		addValue(parametersMap, "firstConsecutiveStaticIP", properties.MasterProfile.FirstConsecutiveStaticIP)
		addValue(parametersMap, "masterVMSize", properties.MasterProfile.VMSize)
		if properties.MasterProfile.HasAvailabilityZones() {
			addValue(parametersMap, "availabilityZones", properties.MasterProfile.AvailabilityZones)
		}
	}

	if linuxProfile != nil {
		addValue(parametersMap, "sshRSAPublicKey", linuxProfile.SSH.PublicKeys[0].KeyData)
		for i, s := range linuxProfile.Secrets {
			addValue(parametersMap, fmt.Sprintf("linuxKeyVaultID%d", i), s.SourceVault.ID)
			for j, c := range s.VaultCertificates {
				addValue(parametersMap, fmt.Sprintf("linuxKeyVaultID%dCertificateURL%d", i, j), c.CertificateURL)
			}
		}
	}

	// Kubernetes Parameters
	assignKubernetesParameters(properties, parametersMap, cloudSpecConfig, generatorCode)

	// Agent parameters
	for _, agentProfile := range properties.AgentPoolProfiles {
		addValue(parametersMap, fmt.Sprintf("%sCount", agentProfile.Name), agentProfile.Count)
		addValue(parametersMap, fmt.Sprintf("%sVMSize", agentProfile.Name), agentProfile.VMSize)
		if agentProfile.HasAvailabilityZones() {
			addValue(parametersMap, fmt.Sprintf("%sAvailabilityZones", agentProfile.Name), agentProfile.AvailabilityZones)
		}
		if agentProfile.IsCustomVNET() {
			addValue(parametersMap, fmt.Sprintf("%sVnetSubnetID", agentProfile.Name), agentProfile.VnetSubnetID)
		} else {
			addValue(parametersMap, fmt.Sprintf("%sSubnet", agentProfile.Name), agentProfile.Subnet)
		}
		if len(agentProfile.Ports) > 0 {
			addValue(parametersMap, fmt.Sprintf("%sEndpointDNSNamePrefix", agentProfile.Name), agentProfile.DNSPrefix)
		}

		if !agentProfile.IsAvailabilitySets() && agentProfile.IsSpotScaleSet() {
			addValue(parametersMap, fmt.Sprintf("%sScaleSetPriority", agentProfile.Name), agentProfile.ScaleSetPriority)
			addValue(parametersMap, fmt.Sprintf("%sScaleSetEvictionPolicy", agentProfile.Name), agentProfile.ScaleSetEvictionPolicy)
		}

		if agentProfile.ImageRef != nil {
			addValue(parametersMap, fmt.Sprintf("%sosImageName", agentProfile.Name), agentProfile.ImageRef.Name)
			addValue(parametersMap, fmt.Sprintf("%sosImageResourceGroup", agentProfile.Name), agentProfile.ImageRef.ResourceGroup)
		}

		// Unless distro is defined, default distro is configured by defaults#setAgentProfileDefaults
		//   Ignores Windows OS
		if !(agentProfile.OSType == api.Windows) {
			addValue(parametersMap, fmt.Sprintf("%sosImageOffer", agentProfile.Name), cloudSpecConfig.OSImageConfig[agentProfile.Distro].ImageOffer)
			addValue(parametersMap, fmt.Sprintf("%sosImageSKU", agentProfile.Name), cloudSpecConfig.OSImageConfig[agentProfile.Distro].ImageSku)
			addValue(parametersMap, fmt.Sprintf("%sosImagePublisher", agentProfile.Name), cloudSpecConfig.OSImageConfig[agentProfile.Distro].ImagePublisher)
			addValue(parametersMap, fmt.Sprintf("%sosImageVersion", agentProfile.Name), cloudSpecConfig.OSImageConfig[agentProfile.Distro].ImageVersion)
		}
	}

	// Windows parameters
	if properties.HasWindows() {
		addValue(parametersMap, "windowsAdminUsername", properties.WindowsProfile.AdminUsername)
		addSecret(parametersMap, "windowsAdminPassword", properties.WindowsProfile.AdminPassword, false)

		if properties.WindowsProfile.HasCustomImage() {
			addValue(parametersMap, "agentWindowsSourceUrl", properties.WindowsProfile.WindowsImageSourceURL)
		} else if properties.WindowsProfile.HasImageRef() {
			addValue(parametersMap, "agentWindowsImageResourceGroup", properties.WindowsProfile.ImageRef.ResourceGroup)
			addValue(parametersMap, "agentWindowsImageName", properties.WindowsProfile.ImageRef.Name)
		} else {
			addValue(parametersMap, "agentWindowsPublisher", properties.WindowsProfile.WindowsPublisher)
			addValue(parametersMap, "agentWindowsOffer", properties.WindowsProfile.WindowsOffer)
			addValue(parametersMap, "agentWindowsSku", properties.WindowsProfile.GetWindowsSku())
			addValue(parametersMap, "agentWindowsVersion", properties.WindowsProfile.ImageVersion)

		}

		addValue(parametersMap, "windowsDockerVersion", properties.WindowsProfile.GetWindowsDockerVersion())

		for i, s := range properties.WindowsProfile.Secrets {
			addValue(parametersMap, fmt.Sprintf("windowsKeyVaultID%d", i), s.SourceVault.ID)
			for j, c := range s.VaultCertificates {
				addValue(parametersMap, fmt.Sprintf("windowsKeyVaultID%dCertificateURL%d", i, j), c.CertificateURL)
				addValue(parametersMap, fmt.Sprintf("windowsKeyVaultID%dCertificateStore%d", i, j), c.CertificateStore)
			}
		}

		addValue(parametersMap, "defaultContainerdRuntimeHandler", properties.WindowsProfile.GetWindowsDefaultRuntimeHandler())
		addValue(parametersMap, "hypervRuntimeHandlers", properties.WindowsProfile.GetWindowsHypervRuntimeHandlers())
	}

	for _, extension := range properties.ExtensionProfiles {
		if extension.ExtensionParametersKeyVaultRef != nil {
			addKeyvaultReference(parametersMap, fmt.Sprintf("%sParameters", extension.Name),
				extension.ExtensionParametersKeyVaultRef.VaultID,
				extension.ExtensionParametersKeyVaultRef.SecretName,
				extension.ExtensionParametersKeyVaultRef.SecretVersion)
		} else {
			addValue(parametersMap, fmt.Sprintf("%sParameters", extension.Name), extension.ExtensionParameters)
		}
	}

	return parametersMap
}
