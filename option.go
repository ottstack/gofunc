package gofunc

type funcOption struct {
	summary string
	tags    []string
}

type applyFunc func(opt *funcOption)

func WithSummary(s string) applyFunc {
	return func(opt *funcOption) {
		opt.summary = s
	}
}

func WithTag(s string) applyFunc {
	return func(opt *funcOption) {
		opt.tags = append(opt.tags, s)
	}
}
