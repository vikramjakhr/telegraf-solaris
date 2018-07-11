package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"errors"
	"strconv"
	"math"
)

var (
	// Default input plugins
	inputDefaults = []string{"cpu", "mem", "swap", "system", "kernel",
							 "processes", "disk", "diskio"}

	// Default output plugins
	outputDefaults = []string{"influxdb"}

	// envVarRe is a regex to find environment variables in the config file
	envVarRe = regexp.MustCompile(`\$\w+`)

	envVarEscaper = strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	)
)

// Config specifies the URL/user/password for the database that telegraf
// will be logging to, as well as all the plugins that the user has
// specified
type Config struct {
	Tags          map[string]string
	InputFilters  []string
	OutputFilters []string

	Agent   *AgentConfig
	Inputs  []*RunningInput
	Outputs []*RunningOutput
}

func NewConfig() *Config {
	c := &Config{
		// Agent defaults:
		Agent: &AgentConfig{
			Interval:      Duration{Duration: 10 * time.Second},
			RoundInterval: true,
			FlushInterval: Duration{Duration: 10 * time.Second},
		},

		Tags:          make(map[string]string),
		Inputs:        make([]*RunningInput, 0),
		Outputs:       make([]*RunningOutput, 0),
		InputFilters:  make([]string, 0),
		OutputFilters: make([]string, 0),
	}
	return c
}

type AgentConfig struct {
	// Interval at which to gather information
	Interval Duration

	RoundInterval       bool
	Precision           Duration
	CollectionJitter    Duration
	FlushInterval       Duration
	FlushJitter         Duration
	MetricBatchSize     int
	MetricBufferLimit   int
	FlushBufferWhenFull bool
	UTC                 bool `toml:"utc"`
	Debug               bool
	Logfile             string
	Quiet               bool
	Hostname            string
	OmitHostname        bool
}

// ListTags returns a string of tags specified in the config,
// line-protocol style
func (c *Config) ListTags() string {
	var tags []string

	for k, v := range c.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(tags)

	return strings.Join(tags, " ")
}

var header = `# Telegraf Configuration
#
# Telegraf is entirely plugin driven. All metrics are gathered from the
# declared inputs, and sent to the declared outputs.
#
# Plugins must be declared in here to be active.
# To deactivate a plugin, comment out the name and any variables.
#
# Use 'telegraf -config telegraf.conf -test' to see what metrics a config
# file would generate.
#
# Environment variables can be used anywhere in this config file, simply prepend
# them with $. For strings the variable must be within quotes (ie, "$STR_VAR"),
# for numbers and booleans they should be plain (ie, $INT_VAR, $BOOL_VAR)


# Global tags can be specified here in key="value" format.
[global_tags]
  # dc = "us-east-1" # will tag all metrics with dc=us-east-1
  # rack = "1a"
  ## Environment variables can be used as tags, and throughout the config file
  # user = "$USER"


# Configuration for telegraf agent
[agent]
  ## Default data collection interval for all inputs
  interval = "10s"
  ## Rounds collection interval to 'interval'
  ## ie, if interval="10s" then always collect on :00, :10, :20, etc.
  round_interval = true

  ## Telegraf will send metrics to outputs in batches of at most
  ## metric_batch_size metrics.
  ## This controls the size of writes that Telegraf sends to output plugins.
  metric_batch_size = 1000

  ## For failed writes, telegraf will cache metric_buffer_limit metrics for each
  ## output, and will flush this buffer on a successful write. Oldest metrics
  ## are dropped first when this buffer fills.
  ## This buffer only fills when writes fail to output plugin(s).
  metric_buffer_limit = 10000

  ## Collection jitter is used to jitter the collection by a random amount.
  ## Each plugin will sleep for a random time within jitter before collecting.
  ## This can be used to avoid many plugins querying things like sysfs at the
  ## same time, which can have a measurable effect on the system.
  collection_jitter = "0s"

  ## Default flushing interval for all outputs. You shouldn't set this below
  ## interval. Maximum flush_interval will be flush_interval + flush_jitter
  flush_interval = "10s"
  ## Jitter the flush interval by a random amount. This is primarily to avoid
  ## large write spikes for users running a large number of telegraf instances.
  ## ie, a jitter of 5s and interval 10s means flushes will happen every 10-15s
  flush_jitter = "0s"

  ## By default or when set to "0s", precision will be set to the same
  ## timestamp order as the collection interval, with the maximum being 1s.
  ##   ie, when interval = "10s", precision will be "1s"
  ##       when interval = "250ms", precision will be "1ms"
  ## Precision will NOT be used for service inputs. It is up to each individual
  ## service input to set the timestamp at the appropriate precision.
  ## Valid time units are "ns", "us" (or "µs"), "ms", "s".
  precision = ""

  ## Logging configuration:
  ## Run telegraf with debug log messages.
  debug = false
  ## Run telegraf in quiet mode (error log messages only).
  quiet = false
  ## Specify the log file name. The empty string means to log to stderr.
  logfile = ""

  ## Override default hostname, if empty use os.Hostname()
  hostname = ""
  ## If set to true, do no set the "host" tag in the telegraf agent.
  omit_hostname = false


###############################################################################
#                            OUTPUT PLUGINS                                   #
###############################################################################
`

var processorHeader = `

###############################################################################
#                            PROCESSOR PLUGINS                                #
###############################################################################
`

var aggregatorHeader = `

###############################################################################
#                            AGGREGATOR PLUGINS                               #
###############################################################################
`

var inputHeader = `

###############################################################################
#                            INPUT PLUGINS                                    #
###############################################################################
`

var serviceInputHeader = `

###############################################################################
#                            SERVICE INPUT PLUGINS                            #
###############################################################################
`

// Inputs returns a list of strings of the configured inputs.
func (c *Config) InputNames() []string {
	var name []string
	for _, input := range c.Inputs {
		name = append(name, input.Name())
	}
	return name
}

// Outputs returns a list of strings of the configured outputs.
func (c *Config) OutputNames() []string {
	var name []string
	for _, output := range c.Outputs {
		name = append(name, output.Name)
	}
	return name
}

func (c *Config) LoadDirectory(path string) error {
	walkfn := func(thispath string, info os.FileInfo, _ error) error {
		if info == nil {
			log.Printf("W! Telegraf is not permitted to read %s", thispath)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if len(name) < 6 || name[len(name)-5:] != ".conf" {
			return nil
		}
		err := c.LoadConfig(thispath)
		if err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(path, walkfn)
}

// Try to find a default config file at these locations (in order):
//   1. $TELEGRAF_CONFIG_PATH
//   2. $HOME/.telegraf/telegraf.conf
//   3. /etc/telegraf/telegraf.conf
//
func getDefaultConfigPath() (string, error) {
	envfile := os.Getenv("TELEGRAF_CONFIG_PATH")
	homefile := os.ExpandEnv("${HOME}/.telegraf/telegraf.conf")
	etcfile := "/etc/telegraf/telegraf.conf"
	if runtime.GOOS == "windows" {
		etcfile = `C:\Program Files\Telegraf\telegraf.conf`
	}
	for _, path := range []string{envfile, homefile, etcfile} {
		if _, err := os.Stat(path); err == nil {
			log.Printf("I! Using config file: %s", path)
			return path, nil
		}
	}

	// if we got here, we didn't find a file in a default location
	return "", fmt.Errorf("No config file specified, and could not find one"+
		" in $TELEGRAF_CONFIG_PATH, %s, or %s", homefile, etcfile)
}

// LoadConfig loads the given config file and applies it to c
func (c *Config) LoadConfig(path string) error {
	var err error
	if path == "" {
		if path, err = getDefaultConfigPath(); err != nil {
			return err
		}
	}
	tbl, err := parseFile(path)
	if err != nil {
		return fmt.Errorf("Error parsing %s, %s", path, err)
	}

	// Parse tags tables first:
	for _, tableName := range []string{"tags", "global_tags"} {
		if val, ok := tbl.Fields[tableName]; ok {
			subTable, ok := val.(*Table)
			if !ok {
				return fmt.Errorf("%s: invalid configuration", path)
			}
			if err = UnmarshalTable(subTable, c.Tags); err != nil {
				log.Printf("E! Could not parse [global_tags] config\n")
				return fmt.Errorf("Error parsing %s, %s", path, err)
			}
		}
	}

	// Parse agent table:
	if val, ok := tbl.Fields["agent"]; ok {
		subTable, ok := val.(*Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = UnmarshalTable(subTable, c.Agent); err != nil {
			log.Printf("E! Could not parse [agent] config\n")
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	// Parse all the rest of the plugins:
	for name, val := range tbl.Fields {
		subTable, ok := val.(*Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}

		switch name {
		case "agent", "global_tags", "tags":
		case "outputs":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				// legacy [outputs.influxdb] support
				case *Table:
					if err = c.addOutput(pluginName, pluginSubTable); err != nil {
						return fmt.Errorf("Error parsing %s, %s", path, err)
					}
				case []*Table:
					for _, t := range pluginSubTable {
						if err = c.addOutput(pluginName, t); err != nil {
							return fmt.Errorf("Error parsing %s, %s", path, err)
						}
					}
				default:
					return fmt.Errorf("Unsupported config format: %s, file %s",
						pluginName, path)
				}
			}
		case "inputs", "plugins":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				// legacy [inputs.cpu] support
				case *Table:
					if err = c.addInput(pluginName, pluginSubTable); err != nil {
						return fmt.Errorf("Error parsing %s, %s", path, err)
					}
				case []*Table:
					for _, t := range pluginSubTable {
						if err = c.addInput(pluginName, t); err != nil {
							return fmt.Errorf("Error parsing %s, %s", path, err)
						}
					}
				default:
					return fmt.Errorf("Unsupported config format: %s, file %s",
						pluginName, path)
				}
			}
		default:
			if err = c.addInput(name, subTable); err != nil {
				return fmt.Errorf("Error parsing %s, %s", path, err)
			}
		}
	}
	return nil
}

func (c *Config) addOutput(name string, table *Table) error {
	if len(c.OutputFilters) > 0 && !sliceContains(name, c.OutputFilters) {
		return nil
	}
	creator, ok := Outputs[name]
	if !ok {
		return fmt.Errorf("Undefined but requested output: %s", name)
	}
	output := creator()

	// If the output has a SetSerializer function, then this means it can write
	// arbitrary types of output, so build the serializer and set it.
	switch t := output.(type) {
	case SerializerOutput:
		serializer, err := buildSerializer(name, table)
		if err != nil {
			return err
		}
		t.SetSerializer(serializer)
	}

	outputConfig, err := buildOutput(name, table)
	if err != nil {
		return err
	}

	if err := UnmarshalTable(table, output); err != nil {
		return err
	}

	ro := NewRunningOutput(name, output, outputConfig,
		c.Agent.MetricBatchSize, c.Agent.MetricBufferLimit)
	c.Outputs = append(c.Outputs, ro)
	return nil
}

func (c *Config) addInput(name string, table *Table) error {
	if len(c.InputFilters) > 0 && !sliceContains(name, c.InputFilters) {
		return nil
	}
	// Legacy support renaming io input to diskio
	if name == "io" {
		name = "diskio"
	}

	creator, ok := Inputs[name]
	if !ok {
		return fmt.Errorf("Undefined but requested input: %s", name)
	}
	input := creator()

	// If the input has a SetParser function, then this means it can accept
	// arbitrary types of input, so build the parser and set it.
	switch t := input.(type) {
	case ParserInput:
		parser, err := buildParser(name, table)
		if err != nil {
			return err
		}
		t.SetParser(parser)
	}

	pluginConfig, err := buildInput(name, table)
	if err != nil {
		return err
	}

	if err := UnmarshalTable(table, input); err != nil {
		return err
	}

	rp := NewRunningInput(input, pluginConfig)
	c.Inputs = append(c.Inputs, rp)
	return nil
}

// trimBOM trims the Byte-Order-Marks from the beginning of the file.
// this is for Windows compatibility only.
// see https://github.com/influxdata/telegraf/issues/1378
func trimBOM(f []byte) []byte {
	return bytes.TrimPrefix(f, []byte("\xef\xbb\xbf"))
}

// escapeEnv escapes a value for inserting into a TOML string.
func escapeEnv(value string) string {
	return envVarEscaper.Replace(value)
}

// parseFile loads a TOML configuration from a provided path and
// returns the AST produced from the TOML parser. When loading the file, it
// will find environment variables and replace them.
func parseFile(fpath string) (*Table, error) {
	contents, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	// ugh windows why
	contents = trimBOM(contents)

	// commenting below code for skipping env variables
	env_vars := envVarRe.FindAll(contents, -1)
	for _, env_var := range env_vars {
		env_val, ok := os.LookupEnv(strings.TrimPrefix(string(env_var), "$"))
		if ok {
			env_val = escapeEnv(env_val)
			contents = bytes.Replace(contents, env_var, []byte(env_val), 1)
		}
	}

	return Parse(contents)
}

type InputCreator func() Input

var Inputs = map[string]InputCreator{}

func AddInput(name string, creator InputCreator) {
	Inputs[name] = creator
}

type OutputCreator func() Output

var Outputs = map[string]OutputCreator{}

func AddOutput(name string, creator OutputCreator) {
	Outputs[name] = creator
}

// PrintInputConfig prints the config usage of a single input.
func PrintInputConfig(name string) error {
	if creator, ok := Inputs[name]; ok {
		printConfig(name, creator(), "inputs", false)
	} else {
		return errors.New(fmt.Sprintf("Input %s not found", name))
	}
	return nil
}

// PrintOutputConfig prints the config usage of a single output.
func PrintOutputConfig(name string) error {
	if creator, ok := Outputs[name]; ok {
		printConfig(name, creator(), "outputs", false)
	} else {
		return errors.New(fmt.Sprintf("Output %s not found", name))
	}
	return nil
}

type printer interface {
	Description() string
	SampleConfig() string
}

func printConfig(name string, p printer, op string, commented bool) {
	comment := ""
	if commented {
		comment = "# "
	}
	fmt.Printf("\n%s# %s\n%s[[%s.%s]]", comment, p.Description(), comment,
		op, name)

	config := p.SampleConfig()
	if config == "" {
		fmt.Printf("\n%s  # no configuration\n\n", comment)
	} else {
		lines := strings.Split(config, "\n")
		for i, line := range lines {
			if i == 0 || i == len(lines)-1 {
				fmt.Print("\n")
				continue
			}
			fmt.Print(strings.TrimRight(comment+line, " ") + "\n")
		}
	}
}

// Duration just wraps time.Duration
type Duration struct {
	Duration time.Duration
}

// UnmarshalTOML parses the duration from the TOML config file
func (d *Duration) UnmarshalTOML(b []byte) error {
	var err error
	b = bytes.Trim(b, `'`)

	// see if we can directly convert it
	d.Duration, err = time.ParseDuration(string(b))
	if err == nil {
		return nil
	}

	// Parse string duration, ie, "1s"
	if uq, err := strconv.Unquote(string(b)); err == nil && len(uq) > 0 {
		d.Duration, err = time.ParseDuration(uq)
		if err == nil {
			return nil
		}
	}

	// First try parsing as integer seconds
	sI, err := strconv.ParseInt(string(b), 10, 64)
	if err == nil {
		d.Duration = time.Second * time.Duration(sI)
		return nil
	}
	// Second try parsing as float seconds
	sF, err := strconv.ParseFloat(string(b), 64)
	if err == nil {
		d.Duration = time.Second * time.Duration(sF)
		return nil
	}

	return nil
}

func sliceContains(name string, list []string) bool {
	for _, b := range list {
		if b == name {
			return true
		}
	}
	return false
}

// buildSerializer grabs the necessary entries from the ast.Table for creating
// a serializers.Serializer object, and creates it, which can then be added onto
// an Output object.
func buildSerializer(name string, tbl *Table) (Serializer, error) {
	c := &SerializerConfig{TimestampUnits: time.Duration(1 * time.Second)}

	if node, ok := tbl.Fields["data_format"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.DataFormat = str.Value
			}
		}
	}

	if c.DataFormat == "" {
		c.DataFormat = "influx"
	}

	if node, ok := tbl.Fields["prefix"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.Prefix = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["template"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.Template = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["json_timestamp_units"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				timestampVal, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, fmt.Errorf("Unable to parse json_timestamp_units as a duration, %s", err)
				}
				// now that we have a duration, truncate it to the nearest
				// power of ten (just in case)
				nearest_exponent := int64(math.Log10(float64(timestampVal.Nanoseconds())))
				new_nanoseconds := int64(math.Pow(10.0, float64(nearest_exponent)))
				c.TimestampUnits = time.Duration(new_nanoseconds)
			}
		}
	}

	delete(tbl.Fields, "data_format")
	delete(tbl.Fields, "prefix")
	delete(tbl.Fields, "template")
	delete(tbl.Fields, "json_timestamp_units")
	return NewSerializer(c)
}

// buildOutput parses output specific items from the ast.Table,
// builds the filter and returns an
// models.OutputConfig to be inserted into models.RunningInput
// Note: error exists in the return for future calls that might require error
func buildOutput(name string, tbl *Table) (*OutputConfig, error) {
	oc := &OutputConfig{
		Name: name,
	}
	return oc, nil
}

// buildParser grabs the necessary entries from the ast.Table for creating
// a parsers.Parser object, and creates it, which can then be added onto
// an Input object.
func buildParser(name string, tbl *Table) (Parser, error) {
	c := &ParserConfig{}

	if node, ok := tbl.Fields["data_format"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.DataFormat = str.Value
			}
		}
	}

	// Legacy support, exec plugin originally parsed JSON by default.
	if name == "exec" && c.DataFormat == "" {
		c.DataFormat = "json"
	} else if c.DataFormat == "" {
		c.DataFormat = "influx"
	}

	if node, ok := tbl.Fields["separator"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.Separator = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["templates"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if ary, ok := kv.Value.(*Array); ok {
				for _, elem := range ary.Value {
					if str, ok := elem.(*String); ok {
						c.Templates = append(c.Templates, str.Value)
					}
				}
			}
		}
	}

	if node, ok := tbl.Fields["tag_keys"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if ary, ok := kv.Value.(*Array); ok {
				for _, elem := range ary.Value {
					if str, ok := elem.(*String); ok {
						c.TagKeys = append(c.TagKeys, str.Value)
					}
				}
			}
		}
	}

	if node, ok := tbl.Fields["data_type"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.DataType = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["collectd_auth_file"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.CollectdAuthFile = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["collectd_security_level"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				c.CollectdSecurityLevel = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["collectd_typesdb"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if ary, ok := kv.Value.(*Array); ok {
				for _, elem := range ary.Value {
					if str, ok := elem.(*String); ok {
						c.CollectdTypesDB = append(c.CollectdTypesDB, str.Value)
					}
				}
			}
		}
	}

	c.MetricName = name

	delete(tbl.Fields, "data_format")
	delete(tbl.Fields, "separator")
	delete(tbl.Fields, "templates")
	delete(tbl.Fields, "tag_keys")
	delete(tbl.Fields, "data_type")
	delete(tbl.Fields, "collectd_auth_file")
	delete(tbl.Fields, "collectd_security_level")
	delete(tbl.Fields, "collectd_typesdb")

	return NewParser(c)
}

// buildInput parses input specific items from the ast.Table,
// builds the filter and returns a
// models.InputConfig to be inserted into models.RunningInput
func buildInput(name string, tbl *Table) (*InputConfig, error) {
	cp := &InputConfig{Name: name}
	if node, ok := tbl.Fields["interval"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				dur, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, err
				}

				cp.Interval = dur
			}
		}
	}

	if node, ok := tbl.Fields["name_prefix"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				cp.MeasurementPrefix = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["name_suffix"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				cp.MeasurementSuffix = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["name_override"]; ok {
		if kv, ok := node.(*KeyValue); ok {
			if str, ok := kv.Value.(*String); ok {
				cp.NameOverride = str.Value
			}
		}
	}

	cp.Tags = make(map[string]string)
	if node, ok := tbl.Fields["tags"]; ok {
		if subtbl, ok := node.(*Table); ok {
			if err := UnmarshalTable(subtbl, cp.Tags); err != nil {
				log.Printf("E! Could not parse tags for input %s\n", name)
			}
		}
	}

	delete(tbl.Fields, "name_prefix")
	delete(tbl.Fields, "name_suffix")
	delete(tbl.Fields, "name_override")
	delete(tbl.Fields, "interval")
	delete(tbl.Fields, "tags")
	return cp, nil
}
