package v1

import (
	corev1 "k8s.io/api/core/v1"
)

type ProjectSpec struct {
	// Project name (e.g., production, staging; required).
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Project API keys, used by agents to send telemetry data (required).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	ApiKeys []ApiKeySpec `json:"apiKeys,omitempty"`
	// Notification integrations.
	NotificationIntegrations *NotificationIntegrationsSpec `json:"notificationIntegrations,omitempty"`
	// Application category settings.
	ApplicationCategories []ApplicationCategorySpec `json:"applicationCategories,omitempty"`
	// Custom applications.
	CustomApplications []CustomApplicationSpec `json:"customApplications,omitempty"`
}

type ApiKeySpec struct {
	// Plain-text API key. Must be unique. Prefer using KeySecret for better security.
	Key string `json:"key,omitempty"`
	// Secret with the API key. Created automatically if missing.
	KeySecret *corev1.SecretKeySelector `json:"keySecret,omitempty"`
	// API key description (optional).
	Description string `json:"description,omitempty"`
}

type NotificationIntegrationsSpec struct {
	// The URL of Coroot instance (required). Used for generating links in notifications.
	// +kubebuilder:validation:Pattern="^https?://.+$"
	BaseUrl string `json:"baseURL"`
	// Slack configuration.
	Slack *NotificationIntegrationSlackSpec `json:"slack,omitempty"`
	// Microsoft Teams configuration.
	Teams *NotificationIntegrationTeamsSpec `json:"teams,omitempty"`
	// PagerDuty configuration.
	Pagerduty *NotificationIntegrationPagerdutySpec `json:"pagerduty,omitempty"`
	// Opsgenie configuration.
	Opsgenie *NotificationIntegrationOpsgenieSpec `json:"opsgenie,omitempty"`
	// Webhook configuration.
	Webhook *NotificationIntegrationWebhookSpec `json:"webhook,omitempty"`
}

type NotificationIntegrationSlackSpec struct {
	// Slack Bot User OAuth Token.
	Token string `json:"token,omitempty"`
	// Secret containing the Token.
	TokenSecret *corev1.SecretKeySelector `json:"tokenSecret,omitempty"`
	// Default Slack channel name.
	DefaultChannel string `json:"defaultChannel"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
}

type NotificationIntegrationTeamsSpec struct {
	// MS Teams Webhook URL.
	WebhookURL string `json:"webhookURL,omitempty"`
	// Secret containing the Webhook URL.
	WebhookURLSecret *corev1.SecretKeySelector `json:"webhookURLSecret,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
}

type NotificationIntegrationPagerdutySpec struct {
	// PagerDuty Integration Key.
	IntegrationKey string `json:"integrationKey,omitempty"`
	// Secret containing the Integration Key.
	IntegrationKeySecret *corev1.SecretKeySelector `json:"integrationKeySecret,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
}

type NotificationIntegrationOpsgenieSpec struct {
	// Opsgenie API Key.
	ApiKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	ApiKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	// EU instance of Opsgenie.
	EUInstance bool `json:"euInstance,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
}

type NotificationIntegrationWebhookSpec struct {
	// Webhook URL (required).
	// +kubebuilder:validation:Pattern="^https?://.+$"
	Url string `json:"url"`
	// Whether to skip verification of the Webhook server's TLS certificate.
	TlsSkipVerify bool `json:"tlsSkipVerify,omitempty"`
	// Basic auth credentials.
	BasicAuth *BasicAuthSpec `json:"basicAuth,omitempty"`
	// Custom headers to include in requests.
	CustomHeaders []HeaderSpec `json:"customHeaders,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
	// Incident template (required if `incidents: true`).
	IncidentTemplate string `json:"incidentTemplate,omitempty"`
	// Deployment template (required if `deployments: true`).
	DeploymentTemplate string `json:"deploymentTemplate,omitempty"`
}

type ApplicationCategorySpec struct {
	// Application category name (required).
	Name string `json:"name"`
	// List of glob patterns in the <namespace>/<application_name> format (e.g., "staging/*", "*/mongo-*").
	CustomPatterns []string `json:"customPatterns,omitempty"`
	// Application category notification settings.
	NotificationSettings ApplicationCategoryNotificationSettingsSpec `json:"notificationSettings,omitempty"`
}

type ApplicationCategoryNotificationSettingsSpec struct {
	// Notify of incidents (SLO violation).
	Incidents ApplicationCategoryNotificationSettingsIncidentsSpec `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments ApplicationCategoryNotificationSettingsDeploymentsSpec `json:"deployments,omitempty"`
}

type ApplicationCategoryNotificationSettingsIncidentsSpec struct {
	Enabled   bool                                                  `json:"enabled,omitempty"`
	Slack     *ApplicationCategoryNotificationSettingsSlackSpec     `json:"slack,omitempty"`
	Teams     *ApplicationCategoryNotificationSettingsTeamsSpec     `json:"teams,omitempty"`
	Pagerduty *ApplicationCategoryNotificationSettingsPagerdutySpec `json:"pagerduty,omitempty"`
	Opsgenie  *ApplicationCategoryNotificationSettingsOpsgenieSpec  `json:"opsgenie,omitempty"`
	Webhook   *ApplicationCategoryNotificationSettingsWebhookSpec   `json:"webhook,omitempty"`
}

type ApplicationCategoryNotificationSettingsDeploymentsSpec struct {
	Enabled bool                                                `json:"enabled,omitempty"`
	Slack   *ApplicationCategoryNotificationSettingsSlackSpec   `json:"slack,omitempty"`
	Teams   *ApplicationCategoryNotificationSettingsTeamsSpec   `json:"teams,omitempty"`
	Webhook *ApplicationCategoryNotificationSettingsWebhookSpec `json:"webhook,omitempty"`
}

type ApplicationCategoryNotificationSettingsSlackSpec struct {
	Enabled bool   `json:"enabled,omitempty"`
	Channel string `json:"channel,omitempty"`
}

type ApplicationCategoryNotificationSettingsTeamsSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsPagerdutySpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsOpsgenieSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsWebhookSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type CustomApplicationSpec struct {
	// Custom application name (required).
	Name string `json:"name"`
	// List of glob patterns for <instance_name>.
	InstancePatterns []string `json:"instancePatterns,omitempty"`
}
