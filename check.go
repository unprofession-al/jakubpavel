package main

import (
	"fmt"
	"time"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type Check struct {
	Resolver        string        `yaml:"resolver" json:"resolver"`
	ResolverTimeout time.Duration `yaml:"resolver_timeout" json:"resolver_timeout"`
	Proto           string        `yaml:"proto" json:"proto"`
	Resolve         string        `yaml:"resolve" json:"resolve"`
	Expect          Expect        `yaml:"-" json:"-"`
	ExpectConfig    ExpectConfig  `yaml:"expect_config" json:"expect_config"`
}

type Expect struct {
	AnswerSection     []dns.RR `yaml:"answer_section" json:"answer_section"`
	AuthoritySection  []dns.RR `yaml:"authority_section" json:"authority_section"`
	AdditionalSection []dns.RR `yaml:"additional_section" json:"additional_section"`
}

type Checker struct {
	Checks map[string]Check
}

func NewChecker(checkConfigs map[string]CheckConfig) (Checker, error) {
	checker := Checker{}
	checks := map[string]Check{}
	for cName, cConfig := range checkConfigs {
		ansSec := []dns.RR{}
		for _, e := range cConfig.Expect.AnswerSection {
			rr, err := dns.NewRR(e)
			if err != nil {
				return checker, err
			}
			ansSec = append(ansSec, rr)
		}

		authSec := []dns.RR{}
		for _, e := range cConfig.Expect.AuthoritySection {
			rr, err := dns.NewRR(e)
			if err != nil {
				return checker, err
			}
			authSec = append(authSec, rr)
		}

		addSec := []dns.RR{}
		for _, e := range cConfig.Expect.AdditionalSection {
			rr, err := dns.NewRR(e)
			if err != nil {
				return checker, err
			}
			addSec = append(addSec, rr)
		}

		expect := Expect{
			AnswerSection:     ansSec,
			AuthoritySection:  authSec,
			AdditionalSection: addSec,
		}

		timeoutString := cConfig.ResolverTimeout
		if cConfig.ResolverTimeout == "" {
			timeoutString = "5s"
		}
		timeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return checker, err
		}

		proto := "udp"
		if cConfig.UseTCP {
			proto = "tcp"
		}

		check := Check{
			Resolver:        cConfig.Resolver,
			ResolverTimeout: timeout,
			Proto:           proto,
			Resolve:         cConfig.Resolve,
			Expect:          expect,
			ExpectConfig:    cConfig.Expect,
		}

		checks[cName] = check
	}

	checker.Checks = checks

	return checker, nil
}

func (c Checker) Run() []CheckResult {
	out := []CheckResult{}

	client := new(dns.Client)
	for checkName, check := range c.Checks {
		client.Net = check.Proto
		client.Timeout = check.ResolverTimeout
		result := CheckResult{
			Name:      checkName,
			Timestamp: time.Now(),
			Check:     check,
		}

		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(check.Resolve), dns.TypeA)
		m.RecursionDesired = true

		r, rtt, err := client.Exchange(m, check.Resolver)
		result.response = r
		result.RTT = rtt

		if err != nil {
			result.ErrorString = err.Error()
			result.Error = fmt.Errorf("ERROR: Failed to run check '%s', error was: %s\n", checkName, err.Error())
			out = append(out, result)
			continue
		}

		if r.Rcode != dns.RcodeSuccess {
			err = fmt.Errorf("ERROR: invalid answer for check '%s'\n", checkName)
			result.ErrorString = err.Error()
			result.Error = err
			out = append(out, result)
			continue
		}

		ansOk := verifyExpectation(check.Expect.AnswerSection, r.Answer)
		authOk := verifyExpectation(check.Expect.AuthoritySection, r.Ns)
		addOk := verifyExpectation(check.Expect.AdditionalSection, r.Extra)
		result.AsExpected = ansOk && authOk && addOk
		out = append(out, result)
	}
	return out
}

type CheckResult struct {
	Name        string        `yaml:"name" json:"name"`
	Timestamp   time.Time     `yaml:"timestamp" json:"timestamp"`
	RTT         time.Duration `yaml:"rtt" json:"rtt"`
	Error       error         `yaml:"-" json:"-"`
	ErrorString string        `yaml:"error_string" json:"error_string"`
	AsExpected  bool          `yaml:"as_expected" json:"as_expected"`
	Check       Check         `yaml:"check" json:"check"`

	response *dns.Msg `yaml:"-" json:"-"`
}

func (cr CheckResult) String() string {
	d, _ := yaml.Marshal(&cr)
	return fmt.Sprintf("--- Metadata:\n%s\n\n--- Response:\n%s\n\n", string(d), cr.response)
}

func (cr CheckResult) OK() bool {
	return cr.Error == nil && cr.AsExpected
}

func verifyExpectation(expected, have []dns.RR) bool {
	for _, expectedRR := range expected {
		found := false
		for _, haveRR := range have {
			if haveRR.String() == expectedRR.String() {
				found = true
				continue
			}
		}
		if !found {
			return false
		}
	}
	return true
}
