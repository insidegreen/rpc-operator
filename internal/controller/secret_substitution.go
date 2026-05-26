package controller

import (
	"regexp"
	"strings"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// varPattern matches ${VARNAME} and ${VARNAME:any_existing_default}.
// Bloblang expressions like ${!...} do not match because '!' is not [A-Za-z_].
var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::[^}]*)?\}`)

// substituteSecrets rewrites ${ENVVAR} and ${ENVVAR:old_default} to ${ENVVAR:actual_value}
// for every envVar listed in refs whose name appears as a key in values.
// Variables not present in values and Bloblang expressions are left unchanged.
// values is built by fetchSecretValues and discarded after the stream PUT — never logged.
func substituteSecrets(yamlText string, refs []rpcv1alpha1.SecretRef, values map[string]string) string {
	if len(refs) == 0 || len(values) == 0 {
		return yamlText
	}
	return varPattern.ReplaceAllStringFunc(yamlText, func(match string) string {
		// match is "${VARNAME}" or "${VARNAME:old}"; strip ${ and }
		inner := match[2 : len(match)-1]
		name := inner
		if idx := strings.IndexByte(inner, ':'); idx >= 0 {
			name = inner[:idx]
		}
		val, ok := values[name]
		if !ok {
			return match
		}
		return "${" + name + ":" + val + "}"
	})
}
