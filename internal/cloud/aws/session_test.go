package aws

import (
	"sync"
	"testing"
	"time"
)

// ============================================================
// CONCEPT: Session Manager Testing
// 🎓 SessionManager implements connection pooling pattern.
// We test initialization, caching behavior, concurrent access,
// and resource cleanup to ensure no memory leaks or race conditions.
// ============================================================

// TestNewSessionManager tests the creation of a new session manager
func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()

	if sm == nil {
		t.Fatal("NewSessionManager() returned nil")
	}

	if sm.sessions == nil {
		t.Error("sessions map not initialized")
	}

	if sm.ec2Clients == nil {
		t.Error("ec2Clients map not initialized")
	}

	// Verify empty initialization
	stats := sm.GetStats()
	if stats["ssm_clients"] != 0 {
		t.Errorf("New SessionManager should have 0 SSM clients, but has %d", stats["ssm_clients"])
	}
	if stats["ec2_clients"] != 0 {
		t.Errorf("New SessionManager should have 0 EC2 clients, but has %d", stats["ec2_clients"])
	}
	if stats["total"] != 0 {
		t.Errorf("New SessionManager should have total=0, but has %d", stats["total"])
	}
}

// ============================================================
// CONCEPT: Key Generation Testing
// 🎓 The makeKey function creates unique identifiers for caching.
// Format: "profile-region" ensures each combination is cached separately.
// ============================================================

// TestMakeKey tests the key generation for session caching
func TestMakeKey(t *testing.T) {
	sm := NewSessionManager()

	tests := []struct {
		name     string
		profile  string
		region   string
		expected string
	}{
		{
			name:     "standard AWS profile and region",
			profile:  "default",
			region:   "us-east-1",
			expected: "default-us-east-1",
		},
		{
			name:     "custom profile",
			profile:  "production",
			region:   "sa-east-1",
			expected: "production-sa-east-1",
		},
		{
			name:     "staging environment",
			profile:  "staging",
			region:   "us-west-2",
			expected: "staging-us-west-2",
		},
		{
			name:     "empty profile and region",
			profile:  "",
			region:   "",
			expected: "-",
		},
		{
			name:     "empty profile only",
			profile:  "",
			region:   "us-east-1",
			expected: "-us-east-1",
		},
		{
			name:     "empty region only",
			profile:  "default",
			region:   "",
			expected: "default-",
		},
		{
			name:     "profile with special characters",
			profile:  "my-prod-profile",
			region:   "eu-west-1",
			expected: "my-prod-profile-eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := sm.makeKey(tt.profile, tt.region)
			if key != tt.expected {
				t.Errorf("makeKey(%q, %q) = %q, want %q", tt.profile, tt.region, key, tt.expected)
			}
		})
	}
}

// TestMakeKey_Uniqueness validates that different combinations produce different keys
func TestMakeKey_Uniqueness(t *testing.T) {
	sm := NewSessionManager()

	key1 := sm.makeKey("profile1", "region1")
	key2 := sm.makeKey("profile2", "region1")
	key3 := sm.makeKey("profile1", "region2")
	key4 := sm.makeKey("profile2", "region2")

	// All keys should be unique
	keys := map[string]bool{
		key1: true,
		key2: true,
		key3: true,
		key4: true,
	}

	if len(keys) != 4 {
		t.Errorf("Expected 4 unique keys, but got %d: %v", len(keys), keys)
	}

	// Same profile+region should produce same key
	key5 := sm.makeKey("profile1", "region1")
	if key1 != key5 {
		t.Errorf("Same profile+region should produce same key: %q != %q", key1, key5)
	}
}

// ============================================================
// CONCEPT: Statistics and Monitoring
// 🎓 GetStats() provides visibility into cache state.
// Useful for monitoring, debugging, and capacity planning.
// ============================================================

// TestGetStats tests the statistics reporting
func TestGetStats(t *testing.T) {
	sm := NewSessionManager()

	t.Run("empty session manager", func(t *testing.T) {
		stats := sm.GetStats()

		if stats["ssm_clients"] != 0 {
			t.Errorf("ssm_clients = %d, want 0", stats["ssm_clients"])
		}
		if stats["ec2_clients"] != 0 {
			t.Errorf("ec2_clients = %d, want 0", stats["ec2_clients"])
		}
		if stats["total"] != 0 {
			t.Errorf("total = %d, want 0", stats["total"])
		}
	})

	t.Run("with SSM sessions", func(t *testing.T) {
		sm := NewSessionManager()

		// Manually add sessions for testing
		sm.sessions["profile1-region1"] = &awsSession{
			profile: "profile1",
			region:  "region1",
		}
		sm.sessions["profile2-region2"] = &awsSession{
			profile: "profile2",
			region:  "region2",
		}

		stats := sm.GetStats()

		if stats["ssm_clients"] != 2 {
			t.Errorf("ssm_clients = %d, want 2", stats["ssm_clients"])
		}
		if stats["total"] != 2 {
			t.Errorf("total = %d, want 2", stats["total"])
		}
	})

	t.Run("with mixed clients", func(t *testing.T) {
		sm := NewSessionManager()

		// Add both SSM and EC2 clients
		sm.sessions["profile1-region1"] = &awsSession{
			profile: "profile1",
			region:  "region1",
		}
		// Note: EC2 client is just *ec2.Client, we can't create it without real AWS
		// This test validates the counting logic works

		stats := sm.GetStats()

		if stats["ssm_clients"] != 1 {
			t.Errorf("ssm_clients = %d, want 1", stats["ssm_clients"])
		}
		// Total should be sum of both
		expectedTotal := stats["ssm_clients"] + stats["ec2_clients"]
		if stats["total"] != expectedTotal {
			t.Errorf("total = %d, want %d (sum of ssm_clients and ec2_clients)", stats["total"], expectedTotal)
		}
	})
}

// ============================================================
// CONCEPT: Resource Cleanup Testing
// 🎓 Close() must properly clean up all cached resources
// to prevent memory leaks in long-running applications.
// ============================================================

// TestClose tests the cleanup functionality
func TestClose(t *testing.T) {
	sm := NewSessionManager()

	// Add some test sessions
	sm.sessions["test1-us-east-1"] = &awsSession{
		profile: "test1",
		region:  "us-east-1",
	}
	sm.sessions["test2-us-west-2"] = &awsSession{
		profile: "test2",
		region:  "us-west-2",
	}

	// Verify sessions exist before close
	statsBefore := sm.GetStats()
	if statsBefore["ssm_clients"] != 2 {
		t.Errorf("Before Close(): ssm_clients = %d, want 2", statsBefore["ssm_clients"])
	}

	// Close and verify cleanup
	sm.Close()

	statsAfter := sm.GetStats()
	if statsAfter["ssm_clients"] != 0 {
		t.Errorf("After Close(): ssm_clients = %d, want 0", statsAfter["ssm_clients"])
	}
	if statsAfter["ec2_clients"] != 0 {
		t.Errorf("After Close(): ec2_clients = %d, want 0", statsAfter["ec2_clients"])
	}
	if statsAfter["total"] != 0 {
		t.Errorf("After Close(): total = %d, want 0", statsAfter["total"])
	}
}

// TestClose_MultipleCallsSafe tests that calling Close() multiple times is safe
func TestClose_MultipleCallsSafe(t *testing.T) {
	sm := NewSessionManager()

	// Add a session
	sm.sessions["test-us-east-1"] = &awsSession{
		profile: "test",
		region:  "us-east-1",
	}

	// Call Close() multiple times - should not panic
	sm.Close()
	sm.Close()
	sm.Close()

	// Verify still clean
	stats := sm.GetStats()
	if stats["total"] != 0 {
		t.Errorf("After multiple Close() calls: total = %d, want 0", stats["total"])
	}
}

// ============================================================
// CONCEPT: Concurrent Access Testing
// 🎓 SessionManager uses sync.RWMutex for thread-safety.
// We test that concurrent reads/writes don't cause race conditions
// or deadlocks. Run with `go test -race` to detect issues.
// ============================================================

// TestConcurrentGetStats tests concurrent access to GetStats
func TestConcurrentGetStats(t *testing.T) {
	sm := NewSessionManager()

	// Add some sessions
	sm.sessions["profile1-us-east-1"] = &awsSession{
		profile: "profile1",
		region:  "us-east-1",
	}

	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Multiple goroutines reading stats concurrently
	for range concurrency {
		go func() {
			defer wg.Done()
			_ = sm.GetStats()
		}()
	}

	// Wait with timeout to detect deadlocks
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Concurrent GetStats completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: possible deadlock in concurrent GetStats")
	}
}

// TestConcurrentClose tests concurrent Close calls
func TestConcurrentClose(t *testing.T) {
	sm := NewSessionManager()

	// Add sessions
	for range 10 {
		key := sm.makeKey("profile", "region")
		sm.sessions[key] = &awsSession{
			profile: "profile",
			region:  "region",
		}
	}

	concurrency := 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Multiple goroutines calling Close concurrently
	for range concurrency {
		go func() {
			defer wg.Done()
			sm.Close()
		}()
	}

	// Wait with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Concurrent Close completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: possible deadlock in concurrent Close")
	}

	// Verify clean state
	stats := sm.GetStats()
	if stats["total"] != 0 {
		t.Errorf("After concurrent Close: total = %d, want 0", stats["total"])
	}
}

// TestConcurrentMixedOperations tests concurrent reads and writes
func TestConcurrentMixedOperations(t *testing.T) {
	sm := NewSessionManager()

	concurrency := 20
	var wg sync.WaitGroup
	wg.Add(concurrency * 2) // readers + writers

	// Concurrent readers
	for range concurrency {
		go func() {
			defer wg.Done()
			for range 10 {
				_ = sm.GetStats()
			}
		}()
	}

	// Concurrent writers (simulated by adding/removing sessions)
	for range concurrency {
		go func() {
			defer wg.Done()
			for range 5 {
				sm.mu.Lock()
				key := sm.makeKey("profile", "region")
				sm.sessions[key] = &awsSession{
					profile: "profile",
					region:  "region",
				}
				sm.mu.Unlock()

				// Small delay
				time.Sleep(1 * time.Millisecond)

				sm.mu.Lock()
				delete(sm.sessions, key)
				sm.mu.Unlock()
			}
		}()
	}

	// Wait with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Concurrent mixed operations completed successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout: possible deadlock in concurrent operations")
	}
}

// ============================================================
// CONCEPT: Session Caching Validation
// 🎓 Validates that sessions are properly cached and reused.
// This is the core of the connection pooling pattern.
// ============================================================

// TestSessionCaching tests that sessions are cached correctly
func TestSessionCaching(t *testing.T) {
	sm := NewSessionManager()

	t.Run("session is added to cache", func(t *testing.T) {
		profile := "test-profile"
		region := "us-east-1"
		key := sm.makeKey(profile, region)

		// Manually add session
		testSession := &awsSession{
			profile: profile,
			region:  region,
		}
		sm.sessions[key] = testSession

		// Verify it's in cache
		stats := sm.GetStats()
		if stats["ssm_clients"] != 1 {
			t.Errorf("Session not cached: ssm_clients = %d, want 1", stats["ssm_clients"])
		}

		// Verify correct session is retrieved
		cachedSession, exists := sm.sessions[key]
		if !exists {
			t.Error("Session not found in cache")
		}
		if cachedSession.profile != profile {
			t.Errorf("Cached session profile = %q, want %q", cachedSession.profile, profile)
		}
		if cachedSession.region != region {
			t.Errorf("Cached session region = %q, want %q", cachedSession.region, region)
		}
	})

	t.Run("different profile-region combinations are cached separately", func(t *testing.T) {
		sm := NewSessionManager()

		// Add multiple sessions
		sessions := []struct {
			profile string
			region  string
		}{
			{"profile1", "us-east-1"},
			{"profile1", "us-west-2"},
			{"profile2", "us-east-1"},
			{"profile2", "us-west-2"},
		}

		for _, s := range sessions {
			key := sm.makeKey(s.profile, s.region)
			sm.sessions[key] = &awsSession{
				profile: s.profile,
				region:  s.region,
			}
		}

		stats := sm.GetStats()
		if stats["ssm_clients"] != 4 {
			t.Errorf("Should have 4 cached sessions, but has %d", stats["ssm_clients"])
		}

		// Verify each session is retrievable
		for _, s := range sessions {
			key := sm.makeKey(s.profile, s.region)
			session, exists := sm.sessions[key]
			if !exists {
				t.Errorf("Session not found for %s-%s", s.profile, s.region)
			}
			if session.profile != s.profile || session.region != s.region {
				t.Errorf("Wrong session retrieved for key %s", key)
			}
		}
	})
}

// TestSessionManager_Isolation tests that different SessionManager instances are isolated
func TestSessionManager_Isolation(t *testing.T) {
	sm1 := NewSessionManager()
	sm2 := NewSessionManager()

	// Add session to sm1
	sm1.sessions["profile1-us-east-1"] = &awsSession{
		profile: "profile1",
		region:  "us-east-1",
	}

	// Verify sm1 has session but sm2 doesn't
	stats1 := sm1.GetStats()
	stats2 := sm2.GetStats()

	if stats1["ssm_clients"] != 1 {
		t.Errorf("sm1 should have 1 session, but has %d", stats1["ssm_clients"])
	}
	if stats2["ssm_clients"] != 0 {
		t.Errorf("sm2 should have 0 sessions, but has %d", stats2["ssm_clients"])
	}
}

// TestAwsSession_StructFields validates the awsSession struct
func TestAwsSession_StructFields(t *testing.T) {
	session := &awsSession{
		profile: "production",
		region:  "sa-east-1",
	}

	if session.profile != "production" {
		t.Errorf("profile = %q, want %q", session.profile, "production")
	}
	if session.region != "sa-east-1" {
		t.Errorf("region = %q, want %q", session.region, "sa-east-1")
	}

	// ssmClient will be nil in unit tests (requires real AWS SDK initialization)
	if session.ssmClient != nil {
		t.Log("ssmClient is set (unexpected in unit test without AWS)")
	}
}
