// Package entry implements the entry-point logic for a typical HTTP server application,
// including opinionated defaults for things like logging and tracing.
//
// Example usage:
//
//	func main() {
//		app := entry.NewApplication("test")
//		defer app.Stop()
//
//		app.Log().Info("Doing some setup")
//		if err := doSomeSetup(); err != nil {
//			app.Fail("Setup failed", err)
//		}
//
//		h := &somethingThatImplementsHttpHandler{}
//
//		entry.RunServer(app, h, "", 5000)
//	}
package entry