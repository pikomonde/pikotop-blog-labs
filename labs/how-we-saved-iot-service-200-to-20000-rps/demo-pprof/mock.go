package main

import (
	"fmt"
	"strings"
	"time"
)

// Permission represents a single access permission entry
type Permission struct {
	ID          string   `json:"id"`
	Resource    string   `json:"resource"`
	Action      string   `json:"action"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	GrantedAt   string   `json:"granted_at"`
	GrantedBy   string   `json:"granted_by"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
}

// AuditLog represents a single audit event on the user record
type AuditLog struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	Detail    string `json:"detail"`
}

// UserPreferences holds all user-configurable settings
type UserPreferences struct {
	Theme              string            `json:"theme"`
	Language           string            `json:"language"`
	Timezone           string            `json:"timezone"`
	NotifyEmail        bool              `json:"notify_email"`
	NotifySMS          bool              `json:"notify_sms"`
	NotifyPush         bool              `json:"notify_push"`
	DashboardLayout    string            `json:"dashboard_layout"`
	DefaultWarehouse   string            `json:"default_warehouse"`
	CustomFields       map[string]string `json:"custom_fields"`
	ReportFrequency    string            `json:"report_frequency"`
	AlertThresholds    map[string]int    `json:"alert_thresholds"`
}

// UserAddress is an address record attached to the user
type UserAddress struct {
	Type       string `json:"type"`
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	IsPrimary  bool   `json:"is_primary"`
}

// DeviceRegistration is a registered device for the user
type DeviceRegistration struct {
	DeviceID    string `json:"device_id"`
	DeviceType  string `json:"device_type"`
	OS          string `json:"os"`
	OSVersion   string `json:"os_version"`
	AppVersion  string `json:"app_version"`
	PushToken   string `json:"push_token"`
	RegisteredAt string `json:"registered_at"`
	LastSeen    string `json:"last_seen"`
	IsActive    bool   `json:"is_active"`
}

// ActivitySummary holds aggregated activity stats
type ActivitySummary struct {
	Period       string         `json:"period"`
	LoginCount   int            `json:"login_count"`
	ActionCount  int            `json:"action_count"`
	ByCategory   map[string]int `json:"by_category"`
	TopResources []string       `json:"top_resources"`
}

// BigUser is the full, bloated user object (~300KB when marshalled)
// This mimics what a real app might store entirely in Redis as a "cache".
type BigUser struct {
	// ── Identity ───────────────────────────────────────────────────
	ID              string `json:"id"`
	ExternalID      string `json:"external_id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	EmailVerified   bool   `json:"email_verified"`
	PhoneNumber     string `json:"phone_number"`
	PhoneVerified   bool   `json:"phone_verified"`
	FullName        string `json:"full_name"`
	DisplayName     string `json:"display_name"`
	AvatarURL       string `json:"avatar_url"`
	Bio             string `json:"bio"`

	// ── Org / Tenant ───────────────────────────────────────────────
	TenantID        string   `json:"tenant_id"`
	TenantName      string   `json:"tenant_name"`
	OrgUnit         string   `json:"org_unit"`
	Department      string   `json:"department"`
	JobTitle        string   `json:"job_title"`
	EmployeeID      string   `json:"employee_id"`
	ManagerID       string   `json:"manager_id"`
	TeamIDs         []string `json:"team_ids"`
	CostCenter      string   `json:"cost_center"`

	// ── Auth metadata ──────────────────────────────────────────────
	Roles           []string `json:"roles"`
	Groups          []string `json:"groups"`
	// THIS is what we actually need — but we're caching the whole BigUser
	Permissions     []Permission `json:"permissions"`

	// ── Timestamps ─────────────────────────────────────────────────
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LastLoginAt     string `json:"last_login_at"`
	LastPasswordChange string `json:"last_password_change"`
	DeactivatedAt   string `json:"deactivated_at,omitempty"`

	// ── Profile detail ─────────────────────────────────────────────
	Addresses       []UserAddress       `json:"addresses"`
	Preferences     UserPreferences     `json:"preferences"`
	Devices         []DeviceRegistration `json:"devices"`

	// ── History & audit ────────────────────────────────────────────
	AuditLogs       []AuditLog          `json:"audit_logs"`
	ActivitySummaries []ActivitySummary  `json:"activity_summaries"`

	// ── Misc large blobs ───────────────────────────────────────────
	// These simulate extra metadata that shouldn't be in cache
	// but ended up here because "it might be useful someday"
	ExtendedProfile map[string]string `json:"extended_profile"`
	TagsMetadata    map[string]string `json:"tags_metadata"`
	Notes           string            `json:"notes"`
	RawConfigBlob   string            `json:"raw_config_blob"`
}

// generateMockUser returns a BigUser with enough data to be ~300KB JSON.
// In a real scenario this might grow even larger with more audit history.
func generateMockUser(userID string) BigUser {
	now := time.Now().UTC().Format(time.RFC3339)

	// ── Permissions: 50 entries ─────────────────────────────────────
	resources := []string{
		"telemetry", "vehicle", "fleet", "user", "device",
		"report", "alert", "config", "firmware", "geofence",
		"dashboard", "billing", "audit", "notification", "api-key",
	}
	actions := []string{"read", "write", "delete", "admin", "export"}
	perms := make([]Permission, 0, 80)
	for i, res := range resources {
		for j, act := range actions {
			perms = append(perms, Permission{
				ID:          fmt.Sprintf("perm-%03d", i*len(actions)+j),
				Resource:    res,
				Action:      act,
				Scope:       "tenant",
				Description: fmt.Sprintf("Allow %s on %s within tenant scope", act, res),
				Tags:        []string{"auto-granted", "v2", res},
				GrantedAt:   now,
				GrantedBy:   "system",
			})
		}
	}

	// ── Audit logs: 200 entries (~biggest chunk) ────────────────────
	eventTypes := []string{"login", "logout", "permission_change", "profile_update", "password_reset", "api_call"}
	ips := []string{"192.168.1.1", "10.0.0.5", "172.16.0.3", "203.0.113.42"}
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_4) Safari/605.1.15",
		"okhttp/4.9.3",
	}
	logs := make([]AuditLog, 200)
	for i := range logs {
		logs[i] = AuditLog{
			EventID:   fmt.Sprintf("evt-%06d", i),
			EventType: eventTypes[i%len(eventTypes)],
			Actor:     userID,
			Timestamp: now,
			IPAddress: ips[i%len(ips)],
			UserAgent: agents[i%len(agents)],
			Detail:    fmt.Sprintf("Event #%d: user performed %s from %s. Session context: tenant-xyz. Request-ID: req-%08x.", i, eventTypes[i%len(eventTypes)], ips[i%len(ips)], i*0xdeadbeef),
		}
	}

	// ── Devices: 10 registered devices ────────────────────────────
	deviceTypes := []string{"mobile", "tablet", "desktop", "iot-gateway"}
	devices := make([]DeviceRegistration, 10)
	for i := range devices {
		devices[i] = DeviceRegistration{
			DeviceID:    fmt.Sprintf("dev-%04d", i),
			DeviceType:  deviceTypes[i%len(deviceTypes)],
			OS:          "Android",
			OSVersion:   "14.0",
			AppVersion:  "3.2.1",
			PushToken:   fmt.Sprintf("fcm-token-%064x", i),
			RegisteredAt: now,
			LastSeen:    now,
			IsActive:    i%3 != 0,
		}
	}

	// ── Extended profile: 200 key-value pairs ──────────────────────
	extProfile := make(map[string]string, 200)
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("profile_field_%03d", i)
		extProfile[key] = fmt.Sprintf("value_%03d_%s", i, strings.Repeat("x", 40))
	}

	// ── Tags metadata: 100 entries ─────────────────────────────────
	tagsMetadata := make(map[string]string, 100)
	for i := 0; i < 100; i++ {
		tagsMetadata[fmt.Sprintf("tag-%02d", i)] = fmt.Sprintf("metadata-value-%d-%s", i, strings.Repeat("m", 30))
	}

	// ── Raw config blob: ~50KB text ────────────────────────────────
	// Simulates a poorly-designed system that stored user-specific
	// config as a large text blob in the same Redis key.
	configLines := make([]string, 500)
	for i := range configLines {
		configLines[i] = fmt.Sprintf("config.section_%03d.key_%03d = value_%s", i/10, i, strings.Repeat("v", 60))
	}
	rawConfig := strings.Join(configLines, "\n")

	// ── Notes: long free-text ──────────────────────────────────────
	noteParts := make([]string, 30)
	for i := range noteParts {
		noteParts[i] = fmt.Sprintf("Note entry %03d: This is a support note added by customer success. The user reported an issue with %s access. Ticket #CS-%05d was created and resolved on %s. Follow-up required: yes.", i, resources[i%len(resources)], i*7+1000, now)
	}
	notes := strings.Join(noteParts, " | ")

	// ── Activity summaries ─────────────────────────────────────────
	activities := []ActivitySummary{
		{
			Period:      "2026-04",
			LoginCount:  42,
			ActionCount: 1337,
			ByCategory:  map[string]int{"telemetry": 800, "vehicle": 300, "report": 237},
			TopResources: []string{"vehicle-001", "fleet-HQ", "config-prod"},
		},
		{
			Period:      "2026-03",
			LoginCount:  38,
			ActionCount: 1102,
			ByCategory:  map[string]int{"telemetry": 650, "vehicle": 280, "alert": 172},
			TopResources: []string{"vehicle-002", "fleet-warehouse-A"},
		},
	}

	// ── Preferences custom fields ──────────────────────────────────
	customFields := make(map[string]string, 50)
	for i := 0; i < 50; i++ {
		customFields[fmt.Sprintf("pref_key_%02d", i)] = fmt.Sprintf("pref_value_%02d", i)
	}
	alertThresholds := map[string]int{
		"cpu_pct": 80, "mem_pct": 85, "disk_pct": 90,
		"rps": 5000, "error_rate_pct": 5,
	}

	return BigUser{
		ID:              userID,
		ExternalID:      "ext-" + userID,
		Username:        "piko.monde",
		Email:           "piko@pikomo.top",
		EmailVerified:   true,
		PhoneNumber:     "+6281234567890",
		PhoneVerified:   true,
		FullName:        "Piko Monde",
		DisplayName:     "Piko",
		AvatarURL:       "https://cdn.pikomo.top/avatars/" + userID + ".jpg",
		Bio:             "Software engineer, IoT enthusiast, and Go developer. Building scalable systems one pprof flamegraph at a time.",
		TenantID:        "tenant-automobile-corp",
		TenantName:      "Automobile Corp",
		OrgUnit:         "engineering",
		Department:      "Platform & Infrastructure",
		JobTitle:        "Senior Software Engineer",
		EmployeeID:      "EMP-00421",
		ManagerID:       "usr-manager-001",
		TeamIDs:         []string{"team-backend", "team-iot", "team-platform", "team-oncall"},
		CostCenter:      "CC-ENGR-PLATFORM",
		Roles:           []string{"engineer", "fleet-admin", "report-viewer", "oncall"},
		Groups:          []string{"all-employees", "engineering", "iot-ops", "backend-guild"},
		Permissions:     perms,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastLoginAt:     now,
		LastPasswordChange: now,
		Addresses: []UserAddress{
			{Type: "office", Street: "Jl. Sudirman No. 1", City: "Jakarta", State: "DKI Jakarta", PostalCode: "10220", Country: "ID", IsPrimary: true},
			{Type: "remote", Street: "Jl. Raya Ubud No. 5", City: "Gianyar", State: "Bali", PostalCode: "80571", Country: "ID", IsPrimary: false},
		},
		Preferences: UserPreferences{
			Theme:            "dark",
			Language:         "en",
			Timezone:         "Asia/Jakarta",
			NotifyEmail:      true,
			NotifySMS:        false,
			NotifyPush:       true,
			DashboardLayout:  "wide",
			DefaultWarehouse: "warehouse-jakarta-01",
			CustomFields:     customFields,
			ReportFrequency:  "weekly",
			AlertThresholds:  alertThresholds,
		},
		Devices:           devices,
		AuditLogs:         logs,
		ActivitySummaries: activities,
		ExtendedProfile:   extProfile,
		TagsMetadata:      tagsMetadata,
		Notes:             notes,
		RawConfigBlob:     rawConfig,
	}
}
