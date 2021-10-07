package types

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/onsi/ginkgo/formatter"
)

type GinkgoError struct {
	Heading      string
	Message      string
	DocLink      string
	CodeLocation CodeLocation
}

func (g GinkgoError) Error() string {
	out := formatter.F("{{bold}}{{red}}%s{{/}}\n", g.Heading)
	if (g.CodeLocation != CodeLocation{}) {
		contentsOfLine := strings.TrimLeft(g.CodeLocation.ContentsOfLine(), "\t ")
		if contentsOfLine != "" {
			out += formatter.F("{{light-gray}}%s{{/}}\n", contentsOfLine)
		}
		out += formatter.F("{{gray}}%s{{/}}\n", g.CodeLocation)
	}
	if g.Message != "" {
		out += formatter.Fiw(1, formatter.COLS, g.Message)
		out += "\n\n"
	}
	if g.DocLink != "" {
		out += formatter.Fiw(1, formatter.COLS, "{{bold}}Learn more at:{{/}} {{cyan}}{{underline}}http://onsi.github.io/ginkgo/#%s{{/}}\n", g.DocLink)
	}

	return out
}

type ginkgoErrors struct{}

var GinkgoErrors = ginkgoErrors{}

func (g ginkgoErrors) UncaughtGinkgoPanic(cl CodeLocation) error {
	return GinkgoError{
		Heading: "Your Test Panicked",
		Message: `When you, or your assertion library, calls Ginkgo's Fail(),
Ginkgo panics to prevent subsequent assertions from running.

Normally Ginkgo rescues this panic so you shouldn't see it.

However, if you make an assertion in a goroutine, Ginkgo can't capture the panic.
To circumvent this, you should call

	defer GinkgoRecover()

at the top of the goroutine that caused this panic.

Alternatively, you may have made an assertion outside of a Ginkgo
leaf node (e.g. in a container node or some out-of-band function) - please move your assertion to
an appropriate Ginkgo node (e.g. a BeforeSuite, BeforeEach, It, etc...).`,
		DocLink:      "marking-specs-as-failed",
		CodeLocation: cl,
	}
}

func (g ginkgoErrors) RerunningSuite() error {
	return GinkgoError{
		Heading: "Rerunning Suite",
		Message: formatter.F(`It looks like you are calling RunSpecs more than once. Ginkgo does not support rerunning suites.  If you want to rerun a suite try {{bold}}ginkgo --repeat=N{{/}} or {{bold}}ginkgo --until-it-fails{{/}}`),
		DocLink: "repeating-test-runs-and-managing-flakey-tests",
	}
}

/* Tree construction errors */

func (g ginkgoErrors) PushingNodeInRunPhase(nodeType NodeType, cl CodeLocation) error {
	return GinkgoError{
		Heading: "Ginkgo detected an issue with your test structure",
		Message: formatter.F(
			`It looks like you are trying to add a {{bold}}[%s]{{/}} node
to the Ginkgo test tree in a leaf node {{bold}}after{{/}} the tests started running.

To enable randomization and parallelization Ginkgo requires the test tree
to be fully construted up front.  In practice, this means that you can
only create nodes like {{bold}}[%s]{{/}} at the top-level or within the
body of a {{bold}}Describe{{/}}, {{bold}}Context{{/}}, or {{bold}}When{{/}}.`, nodeType, nodeType),
		CodeLocation: cl,
		DocLink:      "understanding-ginkgos-lifecycle",
	}
}

func (g ginkgoErrors) CaughtPanicDuringABuildPhase(caughtPanic interface{}, cl CodeLocation) error {
	return GinkgoError{
		Heading: "Assertion or Panic detected during tree construction",
		Message: formatter.F(
			`Ginkgo detected a panic while constructing the test tree.
You may be trying to make an assertion in the body of a container node
(i.e. {{bold}}Describe{{/}}, {{bold}}Context{{/}}, or {{bold}}When{{/}}).

Please ensure all assertions are inside leaf nodes such as {{bold}}BeforeEach{{/}},
{{bold}}It{{/}}, etc.

{{bold}}Here's the content of the panic that was caught:{{/}}
%v`, caughtPanic),
		CodeLocation: cl,
		DocLink:      "do-not-make-assertions-in-container-node-functions",
	}
}

func (g ginkgoErrors) SuiteNodeInNestedContext(nodeType NodeType, cl CodeLocation) error {
	docLink := "global-setup-and-teardown-beforesuite-and-aftersuite"
	if nodeType.Is(NodeTypeReportAfterSuite) {
		docLink = "generating-custom-reports-when-a-test-suite-completes"
	}

	return GinkgoError{
		Heading: "Ginkgo detected an issue with your test structure",
		Message: formatter.F(
			`It looks like you are trying to add a {{bold}}[%s]{{/}} node within a container node.

{{bold}}%s{{/}} can only be called at the top level.`, nodeType, nodeType),
		CodeLocation: cl,
		DocLink:      docLink,
	}
}

func (g ginkgoErrors) SuiteNodeDuringRunPhase(nodeType NodeType, cl CodeLocation) error {
	docLink := "global-setup-and-teardown-beforesuite-and-aftersuite"
	if nodeType.Is(NodeTypeReportAfterSuite) {
		docLink = "generating-custom-reports-when-a-test-suite-completes"
	}

	return GinkgoError{
		Heading: "Ginkgo detected an issue with your test structure",
		Message: formatter.F(
			`It looks like you are trying to add a {{bold}}[%s]{{/}} node within a leaf node after the test started running.

{{bold}}%s{{/}} can only be called at the top level.`, nodeType, nodeType),
		CodeLocation: cl,
		DocLink:      docLink,
	}
}

func (g ginkgoErrors) MultipleBeforeSuiteNodes(nodeType NodeType, cl CodeLocation, earlierNodeType NodeType, earlierCodeLocation CodeLocation) error {
	return ginkgoErrorMultipleSuiteNodes("setup", nodeType, cl, earlierNodeType, earlierCodeLocation)
}

func (g ginkgoErrors) MultipleAfterSuiteNodes(nodeType NodeType, cl CodeLocation, earlierNodeType NodeType, earlierCodeLocation CodeLocation) error {
	return ginkgoErrorMultipleSuiteNodes("teardown", nodeType, cl, earlierNodeType, earlierCodeLocation)
}

func ginkgoErrorMultipleSuiteNodes(setupOrTeardown string, nodeType NodeType, cl CodeLocation, earlierNodeType NodeType, earlierCodeLocation CodeLocation) error {
	return GinkgoError{
		Heading: "Ginkgo detected an issue with your test structure",
		Message: formatter.F(
			`It looks like you are trying to add a {{bold}}[%s]{{/}} node but
you already have a {{bold}}[%s]{{/}} node defined at: {{gray}}%s{{/}}.

Ginkgo only allows you to define one suite %s node.`, nodeType, earlierNodeType, earlierCodeLocation, setupOrTeardown),
		CodeLocation: cl,
		DocLink:      "global-setup-and-teardown-beforesuite-and-aftersuite",
	}
}

/* Decoration errors */
func (g ginkgoErrors) InvalidDecorationForNodeType(cl CodeLocation, nodeType NodeType, decoration string) error {
	return GinkgoError{
		Heading:      "Invalid Decoration",
		Message:      formatter.F(`[%s] node cannot be passed a '%s' decoration`, nodeType, decoration),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) InvalidDeclarationOfFocusedAndPending(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      "Invalid Combination of Decorations: Focused and Pending",
		Message:      formatter.F(`[%s] node was decorated with both Focus and Pending.  At most one is allowed.`, nodeType),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) UnknownDecoration(cl CodeLocation, nodeType NodeType, decoration interface{}) error {
	return GinkgoError{
		Heading:      "Unkown Decoration",
		Message:      formatter.F(`[%s] node was passed an unkown decoration: '%#v'`, nodeType, decoration),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) InvalidBodyType(t reflect.Type, cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading: "Invalid Function",
		Message: formatter.F(`[%s] node must be passed {{bold}}func(){{/}} - i.e. functions that take nothing and return nothing.
You passed {{bold}}%s{{/}} instead.`, nodeType, t),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) MultipleBodyFunctions(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      "Multiple Functions",
		Message:      formatter.F(`[%s] node must be passed a single {{bold}}func(){{/}} - but more than one was passed in.`, nodeType),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) MissingBodyFunction(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      "Missing Functions",
		Message:      formatter.F(`[%s] node must be passed a single {{bold}}func(){{/}} - but none was passed in.`, nodeType),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

/* Ordered Container errors */
func (g ginkgoErrors) InvalidSerialNodeInNonSerialOrderedContainer(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      "Invalid Serial Node in Non-Serial Ordered Container",
		Message:      formatter.F(`[%s] node was decorated with Serial but occurs in an Ordered container that is not marked Serial.  Move the Serial decoration to the outer-most Ordered container to mark all ordered tests within the container as serial.`, nodeType),
		CodeLocation: cl,
		DocLink:      "node-decoration-reference",
	}
}

func (g ginkgoErrors) InvalidNestedOrderedContainer(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "Invalid Nested Ordered Container",
		Message:      "An Ordered container is being nested inside another Ordered container.  This is superfluous and not allowed.  Remove the Ordered decoration from the nested container.",
		CodeLocation: cl,
		DocLink:      "ordered-containers",
	}
}

func (g ginkgoErrors) SetupNodeNotInOrderedContainer(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      "Setup Node not in Ordered Container",
		Message:      fmt.Sprintf("[%s] setup nodes must appear inside an Ordered container.  They cannot be nested within other containers, even containers in an ordered container.", nodeType),
		CodeLocation: cl,
		DocLink:      "ordered-containers",
	}
}

/* DeferCleanup errors */
func (g ginkgoErrors) DeferCleanupInvalidFunction(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "DeferCleanup requires a valid function",
		Message:      "You must pass DeferCleanup a function to invoke.  This function must return zero or one values - if it does return, it must return an error.  The function can take arbitrarily many arguments and you should provide these to DeferCleanup to pass along to the function.",
		CodeLocation: cl,
		DocLink:      "cleaning-up-after-tests",
	}
}

func (g ginkgoErrors) PushingCleanupNodeDuringTreeConstruction(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "DeferCleanup must be called inside a setup or subject node",
		Message:      "You must call DeferCleanup inside a setup node (e.g. BeforeEach, BeforeSuite, AfterAll...) or a subject node (i.e. It).  You can't call DeferCleanup at the top-level or in a container node - use the After* family of setup nodes instead.",
		CodeLocation: cl,
		DocLink:      "cleaning-up-after-tests",
	}
}

func (g ginkgoErrors) PushingCleanupInReportingNode(cl CodeLocation, nodeType NodeType) error {
	return GinkgoError{
		Heading:      fmt.Sprintf("DeferCleanup cannot be called in %s", nodeType),
		Message:      "Please inline your cleanup code - Ginkgo won't run cleanup code after a ReportAfterEach or ReportAfterSuite.",
		CodeLocation: cl,
		DocLink:      "cleaning-up-after-tests",
	}
}

func (g ginkgoErrors) PushingCleanupInCleanupNode(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "DeferCleanup cannot be called in a DeferCleanup callback",
		Message:      "Please inline your cleanup code - Ginkgo doesn't let you call DeferCleanup from within DeferCleanup",
		CodeLocation: cl,
		DocLink:      "cleaning-up-after-tests",
	}
}

/* ReportEntry errors */
func (g ginkgoErrors) TooManyReportEntryValues(cl CodeLocation, arg interface{}) error {
	return GinkgoError{
		Heading:      "Too Many ReportEntry Values",
		Message:      formatter.F(`{{bold}}AddGinkgoReport{{/}} can only be given one value. Got unexpected value: %#v`, arg),
		CodeLocation: cl,
		DocLink:      "attaching-data-to-reports",
	}
}

func (g ginkgoErrors) AddReportEntryNotDuringRunPhase(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "Ginkgo detected an issue with your test structure",
		Message:      formatter.F(`It looks like you are calling {{bold}}AddGinkgoReport{{/}} outside of a running test.  Make sure you call {{bold}}AddGinkgoReport{{/}} inside a runnable node such as It or BeforeEach and not inside the body of a container such as Describe or Context.`),
		CodeLocation: cl,
		DocLink:      "attaching-data-to-reports",
	}
}

/* FileFilter and SkipFilter errors */
func (g ginkgoErrors) InvalidFileFilter(filter string) error {
	return GinkgoError{
		Heading: "Invalid File Filter",
		Message: fmt.Sprintf(`The provided file filter: "%s" is invalid.  File filters must have the format "file", "file:lines" where "file" is a regular expression that will match against the file path and lines is a comma-separated list of integers (e.g. file:1,5,7) or line-ranges (e.g. file:1-3,5-9) or both (e.g. file:1,5-9)`, filter),
		DocLink: "filtering-specs",
	}
}

func (g ginkgoErrors) InvalidFileFilterRegularExpression(filter string, err error) error {
	return GinkgoError{
		Heading: "Invalid File Filter Regular Expression",
		Message: fmt.Sprintf(`The provided file filter: "%s" included an invalid regular expression.  regexp.Compile error: %s`, filter, err),
		DocLink: "filtering-specs",
	}
}

/* Label Errors */
func (g ginkgoErrors) SyntaxErrorParsingLabelFilter(input string, location int, error string) error {
	var message string
	if location >= 0 {
		for i, r := range input {
			if i == location {
				message += "{{red}}{{bold}}{{underline}}"
			}
			message += string(r)
			if i == location {
				message += "{{/}}"
			}
		}
	} else {
		message = input
	}
	message += "\n" + error
	return GinkgoError{
		Heading: "Syntax Error Parsing Label Filter",
		Message: message,
		DocLink: "spec-labels",
	}
}

func (g ginkgoErrors) InvalidLabel(label string, cl CodeLocation) error {
	return GinkgoError{
		Heading:      "Invalid Label",
		Message:      fmt.Sprintf("'%s' is an invalid label.  Labels cannot contain of the following characters: '&|!,()/'", label),
		CodeLocation: cl,
		DocLink:      "spec-labels",
	}
}

func (g ginkgoErrors) InvalidEmptyLabel(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "Invalid Empty Label",
		Message:      "Labels cannot be empty",
		CodeLocation: cl,
		DocLink:      "spec-labels",
	}
}

/* Table errors */
func (g ginkgoErrors) MultipleEntryBodyFunctionsForTable(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "DescribeTable passed multiple functions",
		Message:      "It looks like you are passing multiple functions into DescribeTable.  Only one function can be passed in.  This function will be called for each Entry in the table.",
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

func (g ginkgoErrors) InvalidEntryDescription(cl CodeLocation) error {
	return GinkgoError{
		Heading:      "Invalid Entry description",
		Message:      "Entry description functions must be a string, a function that accepts the entry parameters and returns a string, or nil.",
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

func (g ginkgoErrors) TooFewParametersToTableFunction(expected, actual int, kind string, cl CodeLocation) error {
	return GinkgoError{
		Heading:      fmt.Sprintf("Too few parameters passed in to %s", kind),
		Message:      fmt.Sprintf("The %s expected %d parameters but you passed in %d", kind, expected, actual),
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

func (g ginkgoErrors) TooManyParametersToTableFunction(expected, actual int, kind string, cl CodeLocation) error {
	return GinkgoError{
		Heading:      fmt.Sprintf("Too many parameters passed in to %s", kind),
		Message:      fmt.Sprintf("The %s expected %d parameters but you passed in %d", kind, expected, actual),
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

func (g ginkgoErrors) IncorrectParameterTypeToTableFunction(i int, expected, actual reflect.Type, kind string, cl CodeLocation) error {
	return GinkgoError{
		Heading:      fmt.Sprintf("Incorrect parameters type passed to %s", kind),
		Message:      fmt.Sprintf("The %s expected parameter #%d to be of type <%s> but you passed in <%s>", kind, i, expected, actual),
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

func (g ginkgoErrors) IncorrectVariadicParameterTypeToTableFunction(expected, actual reflect.Type, kind string, cl CodeLocation) error {
	return GinkgoError{
		Heading:      fmt.Sprintf("Incorrect parameters type passed to %s", kind),
		Message:      fmt.Sprintf("The %s expected its variadic parameters to be of type <%s> but you passed in <%s>", kind, expected, actual),
		CodeLocation: cl,
		DocLink:      "table-driven-tests",
	}
}

/* Parallel Synchronization errors */

func (g ginkgoErrors) AggregatedReportUnavailableDueToNodeDisappearing() error {
	return GinkgoError{
		Heading: "Test Report unavailable because a Ginkgo parallel process disappeared",
		Message: "The aggregated report could not be fetched for a ReportAfterSuite node.  A Ginkgo parallel process disappeared before it could finish reporting.",
	}
}

func (g ginkgoErrors) SynchronizedBeforeSuiteFailedOnNode1() error {
	return GinkgoError{
		Heading: "SynchronizedBeforeSuite failed on Ginkgo parallel process #1",
		Message: "The first SynchronizedBeforeSuite function running on Ginkgo parallel process #1 failed.  This test suite will now abort.",
	}
}

func (g ginkgoErrors) SynchronizedBeforeSuiteDisappearedOnNode1() error {
	return GinkgoError{
		Heading: "Node 1 disappeard before SynchronizedBeforeSuite could report back",
		Message: "Ginkgo parallel process #1 disappeared before the first SynchronizedBeforeSuite function completed.  This test suite will now abort.",
	}
}

/* Configuration errors */

var sharedParallelErrorMessage = "It looks like you are trying to run tests in parallel with go test.\nThis is unsupported and you should use the ginkgo CLI instead."

func (g ginkgoErrors) InvalidParallelTotalConfiguration() error {
	return GinkgoError{
		Heading: "-ginkgo.parallel.total must be >= 1",
		Message: sharedParallelErrorMessage,
		DocLink: "parallel-specs",
	}
}

func (g ginkgoErrors) InvalidParallelNodeConfiguration() error {
	return GinkgoError{
		Heading: "-ginkgo.parallel.node is one-indexed and must be <= ginkgo.parallel.total",
		Message: sharedParallelErrorMessage,
		DocLink: "parallel-specs",
	}
}

func (g ginkgoErrors) MissingParallelHostConfiguration() error {
	return GinkgoError{
		Heading: "-ginkgo.parallel.host is missing",
		Message: sharedParallelErrorMessage,
		DocLink: "parallel-specs",
	}
}

func (g ginkgoErrors) UnreachableParallelHost(host string) error {
	return GinkgoError{
		Heading: "Could not reach ginkgo.parallel.host:" + host,
		Message: sharedParallelErrorMessage,
		DocLink: "parallel-specs",
	}
}

func (g ginkgoErrors) DryRunInParallelConfiguration() error {
	return GinkgoError{
		Heading: "Ginkgo only performs -dryRun in serial mode.",
		Message: "Please try running ginkgo -dryRun again, but without -p or -nodes to ensure the test is running in series.",
	}
}

func (g ginkgoErrors) ConflictingVerbosityConfiguration() error {
	return GinkgoError{
		Heading: "Conflicting reporter verbosity settings.",
		Message: "You can't set more than one of -v, -vv and --succinct.  Please pick one!",
	}
}

func (g ginkgoErrors) InvalidGoFlagCount() error {
	return GinkgoError{
		Heading: "Use of go test -count",
		Message: "Ginkgo does not support using go test -count to rerun test suites.  Only -count=1 is allowed.  To repeat test runs, please use the ginkgo cli and `ginkgo -until-it-fails` or `ginkgo -repeat=N`.",
	}
}

func (g ginkgoErrors) InvalidGoFlagParallel() error {
	return GinkgoError{
		Heading: "Use of go test -parallel",
		Message: "Go test's implementation of parallelization does not actually parallelize Ginkgo tests.  Please use the ginkgo cli and `ginkgo -p` or `ginkgo -nodes=N` instead.",
	}
}

func (g ginkgoErrors) BothRepeatAndUntilItFails() error {
	return GinkgoError{
		Heading: "--repeat and --until-it-fails are both set",
		Message: "--until-it-fails directs Ginkgo to rerun tests indefinitely until they fail.  --repeat directs Ginkgo to rerun tests a set number of times.  You can't set both... which would you like?",
	}
}
