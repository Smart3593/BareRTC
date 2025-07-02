package messages_test

import (
	"errors"
	"testing"

	"git.kirsle.net/apps/barertc/pkg/messages"
)

// Boolean representation of the video flags, for testing purposes.
type flags struct {
	Active         bool
	NSFW           bool
	Muted          bool
	IsTalking      bool
	MutualRequired bool
	MutualOpen     bool
	OnlyVIP        bool
}

// Check a video flag integer against the expected bools set on the flags object.
func (f flags) Check(video int) error {
	if video&messages.VideoFlagActive == messages.VideoFlagActive && !f.Active {
		return errors.New("Active expected to be set")
	} else if video&messages.VideoFlagActive != messages.VideoFlagActive && f.Active {
		return errors.New("Active expected NOT to be set")
	}

	if video&messages.VideoFlagNSFW == messages.VideoFlagNSFW && !f.NSFW {
		return errors.New("NSFW expected to be set")
	} else if video&messages.VideoFlagNSFW != messages.VideoFlagNSFW && f.NSFW {
		return errors.New("NSFW expected NOT to be set")
	}

	if video&messages.VideoFlagMuted == messages.VideoFlagMuted && !f.Muted {
		return errors.New("Muted expected to be set")
	} else if video&messages.VideoFlagMuted != messages.VideoFlagMuted && f.Muted {
		return errors.New("Muted expected NOT to be set")
	}

	if video&messages.VideoFlagMutualRequired == messages.VideoFlagMutualRequired && !f.MutualRequired {
		return errors.New("MutualRequired expected to be set")
	} else if video&messages.VideoFlagMutualRequired != messages.VideoFlagMutualRequired && f.MutualRequired {
		return errors.New("MutualRequired expected NOT to be set")
	}

	if video&messages.VideoFlagMutualOpen == messages.VideoFlagMutualOpen && !f.MutualOpen {
		return errors.New("MutualOpen expected to be set")
	} else if video&messages.VideoFlagMutualOpen != messages.VideoFlagMutualOpen && f.MutualOpen {
		return errors.New("MutualOpen expected NOT to be set")
	}

	if video&messages.VideoFlagOnlyVIP == messages.VideoFlagOnlyVIP && !f.OnlyVIP {
		return errors.New("OnlyVIP expected to be set")
	} else if video&messages.VideoFlagOnlyVIP != messages.VideoFlagOnlyVIP && f.OnlyVIP {
		return errors.New("OnlyVIP expected NOT to be set")
	}

	return nil
}

func TestVideoFlag(t *testing.T) {
	type schema struct {
		Flag   int
		Expect flags
	}

	// Tests to run
	var tests = []schema{
		{
			Flag:   0,
			Expect: flags{},
		},
		{
			Flag: 1,
			Expect: flags{
				Active: true,
			},
		},
		{
			Flag: 2,
			Expect: flags{
				NSFW: true,
			},
		},
		{
			Flag: 3,
			Expect: flags{
				Active: true,
				NSFW:   true,
			},
		},
		{
			Flag: 4,
			Expect: flags{
				Muted: true,
			},
		},
		{
			Flag: 5,
			Expect: flags{
				Active: true,
				Muted:  true,
			},
		},
		{
			Flag: 6,
			Expect: flags{
				NSFW:  true,
				Muted: true,
			},
		},
		{
			Flag: 7,
			Expect: flags{
				Active: true,
				NSFW:   true,
				Muted:  true,
			},
		},
		{
			Flag: messages.VideoFlagActive | messages.VideoFlagMuted | messages.VideoFlagMutualRequired | messages.VideoFlagMutualOpen,
			Expect: flags{
				Active:         true,
				Muted:          true,
				MutualRequired: true,
				MutualOpen:     true,
			},
		},
		{
			Flag: messages.VideoFlagActive | messages.VideoFlagOnlyVIP | messages.VideoFlagMutualOpen,
			Expect: flags{
				Active:     true,
				OnlyVIP:    true,
				MutualOpen: true,
			},
		},
		{
			Flag: 32,
			Expect: flags{
				MutualOpen: true,
			},
		},
		{
			Flag: 49,
			Expect: flags{
				Active:         true,
				MutualRequired: true,
				MutualOpen:     true,
			},
		},
	}

	for i, tc := range tests {
		if err := tc.Expect.Check(tc.Flag); err != nil {
			t.Errorf("Test #%d: video flag %d failed check: %s", i, tc.Flag, err)
		}
	}
}

func TestVideoFlagMutation(t *testing.T) {
	// Test bitwise mutations of the video flag.
	var flag int
	var tests = []struct {
		Mutate func(int) int
		Expect flags
	}{
		{
			Mutate: func(v int) int { return 1 },
			Expect: flags{
				Active: true,
			},
		},
		{
			Mutate: func(v int) int {
				return v | messages.VideoFlagMutualOpen
			},
			Expect: flags{
				Active:     true,
				MutualOpen: true,
			},
		},
		{
			Mutate: func(v int) int {
				return v | messages.VideoFlagMutualRequired
			},
			Expect: flags{
				Active:         true,
				MutualOpen:     true,
				MutualRequired: true,
			},
		},
		{
			Mutate: func(v int) int {
				return v | messages.VideoFlagMuted ^ messages.VideoFlagMutualRequired
			},
			Expect: flags{
				Active:     true,
				MutualOpen: true,
				Muted:      true,
			},
		},
		{
			Mutate: func(v int) int {
				return v ^ messages.VideoFlagMutualOpen
			},
			Expect: flags{
				Active: true,
				Muted:  true,
			},
		},
		{
			Mutate: func(v int) int {
				return v | messages.VideoFlagOnlyVIP | messages.VideoFlagNSFW
			},
			Expect: flags{
				Active:  true,
				Muted:   true,
				OnlyVIP: true,
				NSFW:    true,
			},
		},
	}

	for i, tc := range tests {
		flag = tc.Mutate(flag)
		if err := tc.Expect.Check(flag); err != nil {
			t.Errorf("Test #%d: video flag %d failed check: %s", i, flag, err)
		}
	}
}
