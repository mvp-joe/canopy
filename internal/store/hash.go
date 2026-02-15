package store

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ComputeSignatureHash computes a deterministic hash from a symbol's semantic identity.
// Covers: name, kind, visibility, modifiers, type_members, function_params, type_params.
// Location changes do NOT affect the hash.
func ComputeSignatureHash(
	name, kind, visibility string,
	modifiers []string,
	members []*TypeMember,
	params []*FunctionParam,
	typeParams []*TypeParam,
) string {
	h := sha256.New()

	// Core identity.
	fmt.Fprintf(h, "name:%s\n", name)
	fmt.Fprintf(h, "kind:%s\n", kind)
	fmt.Fprintf(h, "visibility:%s\n", visibility)

	// Modifiers — sorted for determinism.
	sorted := make([]string, len(modifiers))
	copy(sorted, modifiers)
	sort.Strings(sorted)
	fmt.Fprintf(h, "modifiers:%s\n", strings.Join(sorted, ","))

	// Type members — sorted by (name, kind) for determinism.
	type memberKey struct{ name, kind, typeExpr, vis string }
	mkeys := make([]memberKey, len(members))
	for i, m := range members {
		mkeys[i] = memberKey{m.Name, m.Kind, m.TypeExpr, m.Visibility}
	}
	sort.Slice(mkeys, func(i, j int) bool {
		if mkeys[i].name != mkeys[j].name {
			return mkeys[i].name < mkeys[j].name
		}
		return mkeys[i].kind < mkeys[j].kind
	})
	for _, mk := range mkeys {
		fmt.Fprintf(h, "member:%s:%s:%s:%s\n", mk.name, mk.kind, mk.typeExpr, mk.vis)
	}

	// Function params — sorted by ordinal.
	type paramKey struct {
		name       string
		ordinal    int
		typeExpr   string
		isReceiver bool
		isReturn   bool
	}
	pkeys := make([]paramKey, len(params))
	for i, p := range params {
		pkeys[i] = paramKey{p.Name, p.Ordinal, p.TypeExpr, p.IsReceiver, p.IsReturn}
	}
	sort.Slice(pkeys, func(i, j int) bool {
		return pkeys[i].ordinal < pkeys[j].ordinal
	})
	for _, pk := range pkeys {
		fmt.Fprintf(h, "param:%s:%d:%s:%v:%v\n", pk.name, pk.ordinal, pk.typeExpr, pk.isReceiver, pk.isReturn)
	}

	// Type params — sorted by ordinal.
	type tpKey struct {
		name        string
		ordinal     int
		variance    string
		paramKind   string
		constraints string
	}
	tkeys := make([]tpKey, len(typeParams))
	for i, tp := range typeParams {
		tkeys[i] = tpKey{tp.Name, tp.Ordinal, tp.Variance, tp.ParamKind, tp.Constraints}
	}
	sort.Slice(tkeys, func(i, j int) bool {
		return tkeys[i].ordinal < tkeys[j].ordinal
	})
	for _, tk := range tkeys {
		fmt.Fprintf(h, "typeparam:%s:%d:%s:%s:%s\n", tk.name, tk.ordinal, tk.variance, tk.paramKind, tk.constraints)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
