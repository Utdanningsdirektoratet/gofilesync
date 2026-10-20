package projects

import (
	"sync"
	"testing"
	"time"

	"github.com/utdanningsdirektoratet/gofilesync/test/utils"
)

// A minimal single-service container setup: one config file plus one source
// directory mounted into the application root, watched via polling.
func TestDockerSingleServiceSetup(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(150, [][3]string{
		{"App config", "src/config.php", "app/config.php"},
		{"App source", "src/app", "app/src"},
	})
	e.Write("src/config.php", "cfg-v1")
	e.Write("src/app/index.php", "index-v1")

	e.StartPoll(100)
	e.WaitForContent("app/config.php", "cfg-v1", 5*time.Second)
	e.WaitForContent("app/src/index.php", "index-v1", 5*time.Second)

	// Edit both the config and a source file.
	e.Write("src/config.php", "cfg-v2")
	e.Write("src/app/index.php", "index-v2")
	e.WaitForContent("app/config.php", "cfg-v2", 5*time.Second)
	e.WaitForContent("app/src/index.php", "index-v2", 5*time.Second)
}

// Concurrent writers across multiple mounted directories (e.g. several services
// rebuilt at once) must all land their final content.
func TestDockerConcurrentServiceWrites(t *testing.T) {
	e := utils.NewEnv(t)
	e.WriteConfig(150, [][3]string{
		{"Service A", "src/a", "app/a"},
		{"Service B", "src/b", "app/b"},
		{"Service C", "src/c", "app/c"},
	})
	for _, s := range []string{"a", "b", "c"} {
		e.Write("src/"+s+"/main.php", "seed")
	}

	e.StartPoll(100)
	e.WaitForContent("app/a/main.php", "seed", 5*time.Second)

	var wg sync.WaitGroup
	for _, s := range []string{"a", "b", "c"} {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				e.Write("src/"+s+"/main.php", s+"-final")
				time.Sleep(3 * time.Millisecond)
			}
		}()
	}
	wg.Wait()

	for _, s := range []string{"a", "b", "c"} {
		e.WaitForContent("app/"+s+"/main.php", s+"-final", 10*time.Second)
	}
}
