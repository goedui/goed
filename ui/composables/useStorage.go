package composables

import (
	"log"

	"github.com/goedui/goed/ui/runtime"
	"github.com/goedui/goed/ui/storage"
)

// UseStorage binds a storage key to a typed reactive ref, mirroring the
// VueUse useStorage contract: the returned ref reads its initial value from
// the store (falling back to def), and every Set writes through to storage so
// state survives restarts. Write failures are logged and do not corrupt the
// in-memory state.
//
//	store, _ := storage.Default()
//	themeName, setThemeName := UseStorage(store, "theme", "Dark")
//	setThemeName("Light") // persists + notifies watchers
func UseStorage[T any](store *storage.Store, key string, def T) (*runtime.Ref[T], func(T)) {
	initial := def
	if store != nil {
		if raw, ok := store.Get(key); ok {
			if typed, ok := raw.(T); ok {
				initial = typed
			}
		}
	}
	value := runtime.NewRef(initial)
	set := func(next T) {
		value.Set(next)
		if store != nil {
			if err := store.Set(key, next); err != nil {
				log.Printf("storage: persist %q: %v", key, err)
			}
		}
	}
	return value, set
}
