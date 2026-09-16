package literacycontract
import "testing"
func TestRegistry(t *testing.T){if len(Registry().Types)!=3{t.Fatal("missing types")};if ValidateResponse("write_char","choice")||Supports("unknown"){t.Fatal("wrong response accepted")};if !ValidateResponse("write_char","handwriting"){t.Fatal("writing missing")}}
