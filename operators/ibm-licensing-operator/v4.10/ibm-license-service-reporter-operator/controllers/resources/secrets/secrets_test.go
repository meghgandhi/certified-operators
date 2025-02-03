//
// Copyright 2023 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package secrets

import (
	"context"
	"flag"
	"strings"
	"testing"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/stretchr/testify/assert"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/mocks"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// BE EXTREMELY CAREFUL WHEN USING FAKE CLIENT AND SECRETS, AS IT DOES NOT WORK WITH StringData and Data FIELDS LIKE THE REAL CLIENT.
// See: https://pkg.go.dev/k8s.io/api@v0.25.6/core/v1#Secret.StringData
// When we create secrets in code we use StringData for convenience, however as it normally is read-only field, during reconciliation
// we use Data field. Fake client does not convert StringData to Data under the hood, so pay extreme attention when you access
// which field in the test. Many hours have been wasted until it became clear, so do not make this effort futile ;)
//
// EDIT: In the scope of the auth code we've switched to the Data field in Secrets for compatibility with unit tests.

func TestReconcileAPISecretToken(t *testing.T) {
	type args struct {
		logger logr.Logger
		config IBMLicenseServiceReporterConfig
	}
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	// first secret won't have token in data
	firstSecret := corev1.Secret{}
	firstSecret.Name = DefaultReporterTokenSecretName
	EnsureCachingLabel(&firstSecret)
	namespace := "test"
	firstSecret.Namespace = namespace
	client := mocks.GetMockClient(&firstSecret)
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Verify secret reconcile passes after default creation",
			args: args{
				logger: logger,
				config: IBMLicenseServiceReporterConfig{
					Instance: reporter,
					Client:   client,
					Scheme:   scheme,
				}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReconcileAPISecretToken(tt.args.logger, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileAPISecretToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			// verify if our client now has StringData or Data filled:
			assert.True(t, KeyExists(firstSecret.Data, ApiReceiverSecretTokenKeyName) ||
				KeyExists(firstSecret.StringData, ApiReceiverSecretTokenKeyName))
			// since secret has StringData it needs to be fixed to data
			firstSecret.Data = make(map[string][]byte)
			firstSecret.Data[ApiReceiverSecretTokenKeyName] = []byte{1, 2, 3}
			err := client.Update(context.TODO(), &firstSecret, nil)
			assert.Nil(t, err)
			// second reconcile should not update secret
			uc := client.UpdateCount
			logger.Info("Starting second loop")
			if err := ReconcileAPISecretToken(tt.args.logger, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileAPISecretToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, uc, client.UpdateCount)
		})
	}
}

func KeyExists[T any](m map[string]T, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// Check if user and password ae generated properly
func TestGeneratePassword(t *testing.T) {
	username, password, err := generateUserAndPassword()
	assert.NoError(t, err)

	assert.Equal(t, "license-administrator", username)
	// Encoded password will always be longer than base one, which is 16
	assert.True(t, 16 < len(password))

	newUsername, newPassword, err := generateUserAndPassword()
	assert.NoError(t, err)

	assert.Equal(t, username, newUsername)
	assert.NotEqual(t, password, newPassword)
}

// Check if htpasswd is generated properly
func TestGenerateHtpasswd(t *testing.T) {
	username := "username"
	password := "password"

	htpasswd, err := generateHtpasswd(username, password)
	assert.NoError(t, err)

	splitHtpassw := strings.Split(htpasswd, ":")
	usr, pswd := splitHtpassw[0], splitHtpassw[1]

	assert.Equal(t, username, usr)
	assert.Equal(t, nil, bcrypt.CompareHashAndPassword([]byte(pswd), []byte(password)))

}

// Check if credentials secret is generated properly
func TestGetReporterCredentialsSecret(t *testing.T) {
	namespace := "ibm-licensing"
	labels := map[string]string{"key": "value"}
	credsSecret, err := GetReporterCredentialsSecret(namespace, labels, map[string]string{})
	assert.NoError(t, err)

	assert.Greater(t, len(credsSecret.Data), 0)
}

// Check if htpasswd secret is generated properly
func TestGetReporterHtpasswdSecret(t *testing.T) {
	namespace := "ibm-licensing"
	labels := map[string]string{"key": "value"}
	username := "admin"
	password := "password"
	credsSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReporterCredentialsSecretName,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeBasicAuth,
		// For the test purposes we use Data instead of String Data, due to mocked client being backed only by a
		// simple object storage (see documentation for StringData field)
		Data: map[string][]byte{"username": []byte(username), "password": []byte(password)},
	}

	htpasswdSecret, err := GetReporterHtpasswdSecret(namespace, labels, map[string]string{}, credsSecret)
	assert.NoError(t, err)

	val, ok := htpasswdSecret.Data["data"]
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	splitSecretHtpasswd := strings.Split(string(htpasswdSecret.Data["data"]), ":")
	secretUsernameHtpasswd := splitSecretHtpasswd[0]
	secretHashedPassword := splitSecretHtpasswd[1]

	assert.Equal(t, username, secretUsernameHtpasswd)
	assert.Equal(t, nil, bcrypt.CompareHashAndPassword([]byte(secretHashedPassword), []byte(password)))
}

// Check if auth secrets are created correctly
func TestReconcileCredentialsSecrets(t *testing.T) {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	namespace := "ibm-licensing"
	reporter.Namespace = namespace
	client := fake.NewClientBuilder().Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCredentialsSecrets(logger, config)
	assert.NoError(t, err)

	foundCredsSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterCredentialsSecretName}, &foundCredsSecret)
	assert.NoError(t, err)

	username, usrOk := foundCredsSecret.Data["username"]
	assert.True(t, usrOk)
	assert.NotEmpty(t, username)

	password, passwdOk := foundCredsSecret.Data["password"]
	assert.True(t, passwdOk)
	assert.NotEmpty(t, password)

	foundHtpasswdSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterHtpasswdSecretName}, &foundHtpasswdSecret)
	assert.NoError(t, err)

	htpasswd, htpasswdOk := foundHtpasswdSecret.Data["data"]
	assert.True(t, htpasswdOk)
	assert.NotEmpty(t, htpasswd)
}

// Check if changed credentials are not overwritten by reconcile loop
func TestReconcileCredentialsSecretsDoNotOverwriteCreds(t *testing.T) {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	namespace := "ibm-licensing"
	reporter.Namespace = namespace
	labels := LabelsForMeta(reporter) // THESE EXACT LABELS MUST BE SET
	expectedUsername := "admin"
	expectedPassword := "password"
	credsSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReporterCredentialsSecretName,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeBasicAuth,
		// For the test purposes we use Data instead of String Data, due to mocked client being backed only by a
		// simple object storage (see documentation for StringData field)
		Data: map[string][]byte{"username": []byte(expectedUsername), "password": []byte(expectedPassword)},
	}

	client := fake.NewClientBuilder().WithObjects(&credsSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCredentialsSecrets(logger, config)
	assert.NoError(t, err)

	foundCredsSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterCredentialsSecretName}, &foundCredsSecret)
	assert.NoError(t, err)

	username, usrOk := foundCredsSecret.Data["username"]
	assert.True(t, usrOk)
	assert.NotEmpty(t, username)

	assert.Equal(t, expectedUsername, string(username))

	password, passwdOk := foundCredsSecret.Data["password"]
	assert.True(t, passwdOk)
	assert.NotEmpty(t, password)

	assert.Equal(t, expectedPassword, string(password))
}

func TestReconcileCredentialsSecretsPropagateNewCreds(t *testing.T) {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	labels := LabelsForMeta(reporter)
	namespace := "ibm-licensing"
	expectedUsername := "admin"
	expectedPassword := "password"
	reporter.Namespace = namespace

	// We assume function was already tested earlier and works
	username, password, err := generateUserAndPassword()
	assert.NoError(t, err)

	credsSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReporterCredentialsSecretName,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{"username": []byte(username), "password": []byte(password)},
	}

	// We assume function was already tested earlier and works
	htpasswd, err := generateHtpasswd(username, password)
	assert.NoError(t, err)

	htpasswdSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ReporterHtpasswdSecretName,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"data": []byte(htpasswd)},
	}

	client := fake.NewClientBuilder().WithObjects(&credsSecret, &htpasswdSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err = ReconcileCredentialsSecrets(logger, config)
	assert.NoError(t, err)

	foundCredsSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterCredentialsSecretName}, &foundCredsSecret)
	assert.NoError(t, err)

	foundUsername, usrOk := foundCredsSecret.Data["username"]
	assert.True(t, usrOk)
	assert.NotEmpty(t, foundUsername)

	foundPassword, passwdOk := foundCredsSecret.Data["password"]
	assert.True(t, passwdOk)
	assert.NotEmpty(t, foundPassword)

	foundHtpasswdSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterHtpasswdSecretName}, &foundHtpasswdSecret)
	assert.NoError(t, err)

	// Check if generated creds are different than expected
	assert.Equal(t, username, string(foundUsername))
	assert.Equal(t, password, string(foundPassword))

	// Change credentials to expected
	foundCredsSecret.Data["username"] = []byte(expectedUsername)
	foundCredsSecret.Data["password"] = []byte(expectedPassword)

	err = client.Update(context.TODO(), &foundCredsSecret)
	assert.NoError(t, err)

	// Trigger reconciliation
	err = ReconcileCredentialsSecrets(logger, config)
	assert.NoError(t, err)

	foundReconciledCredsSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ReporterCredentialsSecretName, Namespace: namespace}, &foundReconciledCredsSecret)
	assert.NoError(t, err)

	// Check if changed credentials were not overwritten during reconciliation
	assert.Equal(t, expectedUsername, string(foundReconciledCredsSecret.Data["username"]))
	assert.Equal(t, expectedPassword, string(foundReconciledCredsSecret.Data["password"]))

	foundReconciledHtpasswdSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ReporterHtpasswdSecretName, Namespace: namespace}, &foundReconciledHtpasswdSecret)
	assert.NoError(t, err)

	foundHtpasswd := strings.Split(string(foundReconciledHtpasswdSecret.Data["data"]), ":")
	foundReconciledUsername := foundHtpasswd[0]
	foundReconciledPassword := foundHtpasswd[1]

	// Check if changed credentials were propagated to the htpasswd secret
	assert.Equal(t, expectedUsername, foundReconciledUsername)
	assert.Equal(t, nil, bcrypt.CompareHashAndPassword([]byte(foundReconciledPassword), []byte(expectedPassword)))
}

func TestGetCookieSecret(t *testing.T) {
	namespace := "test"
	metaLabels := mocks.GetLabelsForMeta(mocks.CookieSecretNameMock)
	actualSecret := GetCookieSecret(namespace, metaLabels, map[string]string{})

	assert.Equal(t, mocks.CookieSecretNameMock, actualSecret.Name, "cookie secret should be named "+mocks.CookieSecretNameMock)
	assert.True(t, len(actualSecret.Data) == 1, "cookie secret should contain exactly one key")
	val, found := actualSecret.Data["data"]
	assert.True(t, found, "cookie secret should have the \"data\" key")
	assert.True(t, len(val) > 0, "cookie secret should contain a non-empty cookie")
}

// Check if secret is created if it not exists
func TestReconcileCookieSecretCreateSecret(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName

	client := fake.NewClientBuilder().Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCookieSecret(logger, config)
	assert.NoError(t, err)

	foundSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: mocks.CookieSecretNameMock}, &foundSecret)
	assert.NoError(t, err)

	val, found := foundSecret.Data["data"]
	assert.True(t, len(foundSecret.Data) == 1, "cookie secret should contain exactly one key")
	assert.True(t, found, "secret should have \"data\" key with cookie value")
	assert.True(t, len(val) > 0, "cookie secret should contain a non-empty cookie")
}

// Check if secret is preserved as is if it already exists / was changed
func TestReconcileCookieSecretDoNotOverwriteSecret(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	existingSecret := mocks.CookieSecret

	client := fake.NewClientBuilder().WithObjects(&existingSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCookieSecret(logger, config)
	assert.NoError(t, err)

	foundSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: mocks.CookieSecretNameMock}, &foundSecret)
	assert.NoError(t, err)

	assert.Equal(t, existingSecret.Data, foundSecret.Data)
}

// Check if secret is fixed when cookie value is left empty
func TestReconcileCookieSecretFixSecretEmptyCookie(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	existingSecret := mocks.CookieSecret

	// set cookie value to empty
	existingSecret.Data["data"] = []byte{}

	client := fake.NewClientBuilder().WithObjects(&existingSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCookieSecret(logger, config)
	assert.NoError(t, err)

	foundSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: mocks.CookieSecretNameMock}, &foundSecret)
	assert.NoError(t, err)

	val, found := foundSecret.Data["data"]
	assert.True(t, len(foundSecret.Data) == 1, "cookie secret should contain exactly one key")
	assert.True(t, found, "secret should have \"Data\" key with cookie value")
	assert.True(t, len(val) > 0, "cookie secret should contain a non-empty cookie")
}

// Check if secret is fixed when cookie is under different key
func TestReconcileCookieSecretFixSecretDifferentKey(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	existingSecret := mocks.CookieSecret

	// provide cookie under different key
	existingSecret.Data = map[string][]byte{"not-data": []byte("Wkrh_97IIbroIMxddk5mONsw8wGGDNC-Po_a0olD82k")}

	client := fake.NewClientBuilder().WithObjects(&existingSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCookieSecret(logger, config)
	assert.NoError(t, err)

	foundSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: mocks.CookieSecretNameMock}, &foundSecret)
	assert.NoError(t, err)

	val, found := foundSecret.Data["data"]
	assert.True(t, len(foundSecret.Data) == 1, "cookie secret should contain exactly one key")
	assert.True(t, found, "secret should have \"Data\" key with cookie value")
	assert.True(t, len(val) > 0, "cookie secret should contain a non-empty cookie")
}

// Check if secret has cookie under the data key
func TestReconcileCookieSecretFixEmptySecret(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	existingSecret := mocks.CookieSecret

	// secret is empty
	existingSecret.Data = nil

	client := fake.NewClientBuilder().WithObjects(&existingSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	err := ReconcileCookieSecret(logger, config)
	assert.NoError(t, err)

	foundSecret := corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: mocks.CookieSecretNameMock}, &foundSecret)
	assert.NoError(t, err)

	val, found := foundSecret.Data["data"]
	assert.True(t, len(foundSecret.Data) == 1, "cookie secret should contain exactly one key")
	assert.True(t, found, "secret should have \"Data\" key with cookie value")
	assert.True(t, len(val) > 0, "cookie secret should contain a non-empty cookie")
}

func TestReconcilingSecretsAppliesSpecLabels(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	reporter := operatorv1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	// Basic route and certs source required for ReconcileCertificateSecrets
	reporter.Spec.HTTPSCertsSource = ""
	reporter.Spec.RouteEnabled = ptr.To(true)
	route := routev1.Route{}
	route.Name = LicenseReporterResourceBase
	route.Namespace = namespace
	route.Spec.Host = "test.example.com"
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&route).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance:   reporter,
		Client:     client,
		Scheme:     scheme,
		IsRouteAPI: true,
	}

	// Reconcile secrets should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	for _, reconcileFunction := range []func(_ logr.Logger, _ IBMLicenseServiceReporterConfig) error{
		ReconcileAPISecretToken,
		ReconcileDatabaseSecret,
		ReconcileCredentialsSecrets,
		ReconcileCookieSecret,
		ReconcileInternalCertificateSecret,
		ReconcileExternalCertificateSecret,
	} {
		err := reconcileFunction(logger, config)
		assert.NoError(t, err)
	}

	// Check secrets created with spec.labels
	for _, secretName := range []string{
		DefaultReporterTokenSecretName,
		ReporterCredentialsSecretName,
		ReporterHtpasswdSecretName,
		ReporterAuthCookieSecret,
		DatabaseConfigSecretName,
		ExternalCertName,
	} {
		foundSecret := corev1.Secret{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: secretName}, &foundSecret)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", foundSecret.GetLabels()["test-label"], secretName+" should have spec.labels applied")
	}

	// Reconcile secrets should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	for _, reconcileFunction := range []func(_ logr.Logger, _ IBMLicenseServiceReporterConfig) error{
		ReconcileAPISecretToken,
		ReconcileDatabaseSecret,
		ReconcileCredentialsSecrets,
		ReconcileCookieSecret,
		ReconcileInternalCertificateSecret,
		ReconcileExternalCertificateSecret,
	} {
		err := reconcileFunction(logger, config)
		assert.NoError(t, err)
	}

	// Check secrets patched with spec.labels
	for _, secretName := range []string{
		DefaultReporterTokenSecretName,
		ReporterCredentialsSecretName,
		ReporterHtpasswdSecretName,
		ReporterAuthCookieSecret,
		DatabaseConfigSecretName,
		ExternalCertName,
	} {
		foundSecret := corev1.Secret{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: secretName}, &foundSecret)
		assert.NoError(t, err)
		assert.Equal(t, "test-value-patched", foundSecret.GetLabels()["test-label"], secretName+" should have spec.labels applied")
	}
}
