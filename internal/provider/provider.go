--- a/internal/provider/provider.go
+++ b/internal/provider/provider.go
@@
 func parseARMUseOIDC() bool {
-    switch os.Getenv("ARM_USE_OIDC") {
-    case "1", "true", "yes", "on", "TRUE", "Yes":
-        return true
-    default:
-        return false
-    }
+    switch strings.ToLower(strings.TrimSpace(os.Getenv("ARM_USE_OIDC"))) {
+    case "1", "true", "yes", "on":
+        return true
+    default:
+        return false
+    }
 }
