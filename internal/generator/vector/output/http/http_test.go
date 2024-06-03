package http

import (
	"github.com/openshift/cluster-logging-operator/internal/generator/framework"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	logging "github.com/openshift/cluster-logging-operator/api/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/generator/utils"
	. "github.com/openshift/cluster-logging-operator/test/matchers"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Generate vector config", func() {
	DescribeTable("for Http output", func(output logging.OutputSpec, secret *corev1.Secret, op framework.Options, exp string) {
		conf := New(helpers.FormatComponentID(output.Name), output, []string{"application"}, secret, nil, op) //, includeNS, excludes)
		Expect(exp).To(EqualConfigFrom(conf))
	},
		Entry("",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				Secret: &logging.OutputSecretSpec{
					Name: "http-receiver",
				},
				OutputTypeSpec: logging.OutputTypeSpec{
					Http: &logging.Http{
						Headers: map[string]string{
							"h2": "v2",
							"h1": "v1",
						},
						Method: "POST",
					},
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("username"),
					"password": []byte("password"),
				},
			},
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"

[sinks.http_receiver.request]
headers = {"h1"="v1","h2"="v2"}

# Basic Auth Config
[sinks.http_receiver.auth]
strategy = "basic"
user = "username"
password = "password"
`,
		),
		Entry("with custom bearer token",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				Secret: &logging.OutputSecretSpec{
					Name: "http-receiver",
				},
				OutputTypeSpec: logging.OutputTypeSpec{
					Http: &logging.Http{
						Headers: map[string]string{
							"h2": "v2",
							"h1": "v1",
						},
						Method: "POST",
					},
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					"token": []byte("token-for-custom-http"),
				},
			},
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"

[sinks.http_receiver.request]
headers = {"h1"="v1","h2"="v2"}

# Bearer Auth Config
[sinks.http_receiver.auth]
strategy = "bearer"
token = "token-for-custom-http"
`,
		),
		Entry("with Http config",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				OutputTypeSpec: logging.OutputTypeSpec{Http: &logging.Http{
					Timeout: 50,
					Headers: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				}},
				Secret: &logging.OutputSecretSpec{
					Name: "http-receiver",
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					"token": []byte("token-for-custom-http"),
				},
			},
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"


[sinks.http_receiver.request]
timeout_secs = 50
headers = {"k1"="v1","k2"="v2"}

# Bearer Auth Config
[sinks.http_receiver.auth]
strategy = "bearer"
token = "token-for-custom-http"
`,
		),
		Entry("with Http config",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				OutputTypeSpec: logging.OutputTypeSpec{Http: &logging.Http{
					Timeout: 50,
					Headers: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				}},
				Secret: &logging.OutputSecretSpec{
					Name: "http-receiver",
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					"token": []byte("token-for-custom-http"),
				},
			},
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"

[sinks.http_receiver.request]
timeout_secs = 50
headers = {"k1"="v1","k2"="v2"}

# Bearer Auth Config
[sinks.http_receiver.auth]
strategy = "bearer"
token = "token-for-custom-http"
`,
		),
		Entry("with TLS config",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				OutputTypeSpec: logging.OutputTypeSpec{Http: &logging.Http{
					Timeout: 50,
					Headers: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				}},
				TLS: &logging.OutputTLSSpec{
					InsecureSkipVerify: true,
				},
				Secret: &logging.OutputSecretSpec{
					Name: "http-receiver",
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					"token":         []byte("token-for-custom-http"),
					"tls.crt":       []byte("-- crt-- "),
					"tls.key":       []byte("-- key-- "),
					"ca-bundle.crt": []byte("-- ca-bundle -- "),
				},
			},
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"

[sinks.http_receiver.request]
timeout_secs = 50
headers = {"k1"="v1","k2"="v2"}

[sinks.http_receiver.tls]
verify_certificate = false
verify_hostname = false
key_file = "/var/run/ocp-collector/secrets/http-receiver/tls.key"
crt_file = "/var/run/ocp-collector/secrets/http-receiver/tls.crt"
ca_file = "/var/run/ocp-collector/secrets/http-receiver/ca-bundle.crt"

# Bearer Auth Config
[sinks.http_receiver.auth]
strategy = "bearer"
token = "token-for-custom-http"
`,
		),
		Entry("with TLS.InsecureSkipVerify=true when no secret",
			logging.OutputSpec{
				Type: logging.OutputTypeHttp,
				Name: "http-receiver",
				URL:  "https://my-logstore.com",
				OutputTypeSpec: logging.OutputTypeSpec{Http: &logging.Http{
					Timeout: 50,
					Headers: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				}},
				TLS: &logging.OutputTLSSpec{
					InsecureSkipVerify: true,
				},
			},
			nil,
			framework.NoOptions,
			`
[transforms.http_receiver_normalize]
type = "remap"
inputs = ["application"]
source = '''
  del(.file)
'''

[transforms.http_receiver_dedot]
type = "remap"
inputs = ["http_receiver_normalize"]
source = '''
  .openshift.sequence = to_unix_timestamp(now(), unit: "nanoseconds")
  if exists(.kubernetes.namespace_labels) {
	  for_each(object!(.kubernetes.namespace_labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.namespace_labels = set!(.kubernetes.namespace_labels,[newkey],value)
		if newkey != key {
		  .kubernetes.namespace_labels = remove!(.kubernetes.namespace_labels,[key],true)
		}
	  }
  }
  if exists(.kubernetes.labels) {
	  for_each(object!(.kubernetes.labels)) -> |key,value| { 
		newkey = replace(key, r'[\./]', "_") 
		.kubernetes.labels = set!(.kubernetes.labels,[newkey],value)
		if newkey != key {
		  .kubernetes.labels = remove!(.kubernetes.labels,[key],true)
		}
	  }
  }
'''

[sinks.http_receiver]
type = "http"
inputs = ["http_receiver_dedot"]
uri = "https://my-logstore.com"
method = "post"

[sinks.http_receiver.encoding]
codec = "json"

[sinks.http_receiver.request]
timeout_secs = 50
headers = {"k1"="v1","k2"="v2"}

[sinks.http_receiver.tls]
verify_certificate = false
verify_hostname = false
`,
		),
	)
})

func TestHeaders(t *testing.T) {
	h := map[string]string{
		"k1": "v1",
		"k2": "v2",
	}
	expected := `{"k1"="v1","k2"="v2"}`
	got := utils.ToHeaderStr(h, "%q=%q")
	if got != expected {
		t.Logf("got: %s, expected: %s", got, expected)
		t.Fail()
	}
}

func TestVectorConfGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vector Conf Generation")
}
