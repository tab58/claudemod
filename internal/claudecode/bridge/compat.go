package bridge

// Run is a backward-compatible wrapper that spawns a single session
// and blocks until it exits. Existing callers (main.go) need no changes.
func Run(claudePath string, args []string, cfg Config) (int, error) {
	b, err := New(BridgeConfig{})
	if err != nil {
		return 1, err
	}
	defer b.Close()

	s, err := b.Spawn(claudePath, args, cfg)
	if err != nil {
		return 1, err
	}
	if err := b.Activate(s); err != nil {
		return 1, err
	}
	return s.Wait()
}
