package config_test

import (
	"encoding/json"
	"errors"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-micro-cli/config"
)

var _ = Describe("fileSystemConfigService", func() {
	var (
		service            Service
		configFilePath     string
		deploymentFilePath string
		fakeFs             *fakesys.FakeFileSystem
		stemcells          []StemcellRecord
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
		configFilePath = "/config/file/path"
		deploymentFilePath = "/some/deployment.json"
		logger := boshlog.NewLogger(boshlog.LevelNone)
		service = NewFileSystemConfigService(logger, fakeFs, configFilePath)
	})

	Describe("Load", func() {
		It("reads the given config file", func() {
			stemcells = []StemcellRecord{
				StemcellRecord{
					Name:    "fake-stemcell-name-1",
					Version: "fake-stemcell-version-1",
					SHA1:    "fake-stemcell-sha1-1",
					CID:     "fake-stemcell-cid-1",
				},
				StemcellRecord{
					Name:    "fake-stemcell-name-2",
					Version: "fake-stemcell-version-2",
					SHA1:    "fake-stemcell-sha1-2",
					CID:     "fake-stemcell-cid-2",
				},
			}
			configFileContents, err := json.Marshal(Config{
				Deployment: "/some/manifest.yml",
				Stemcells:  stemcells,
			})
			deploymentFileContents, err := json.Marshal(map[string]interface{}{
				"uuid":      "deadbeef",
				"stemcells": stemcells,
			})
			fakeFs.WriteFile(configFilePath, configFileContents)
			fakeFs.WriteFile(deploymentFilePath, deploymentFileContents)

			config, err := service.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Deployment).To(Equal("/some/manifest.yml"))
			Expect(config.DeploymentUUID).To(Equal("deadbeef"))
			Expect(config.Stemcells).To(Equal(stemcells))
		})

		Context("when the config and deployment file do not exist", func() {
			It("returns an empty Config", func() {
				config, err := service.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(Equal(Config{}))
			})
		})

		Context("when the config exists and deployment file does not exist", func() {
			It("returns the content of the config ", func() {
				configFileContents, err := json.Marshal(Config{
					Deployment: "/some/manifest.yml",
					Stemcells:  stemcells,
				})
				fakeFs.WriteFile(configFilePath, configFileContents)

				config, err := service.Load()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Loading deployment file"))
				Expect(config).To(Equal(Config{}))
			})
		})

		Context("when the config file is invalid", func() {
			It("returns an empty Config and an error", func() {
				fakeFs.WriteFileString(configFilePath, "invalid json")
				config, err := service.Load()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshalling JSON config file `/config/file/path'"))
				Expect(config).To(Equal(Config{}))
			})
		})

		Context("when the deployment file is invalid", func() {
			It("returns an empty Config and an error", func() {
				configFileContents, err := json.Marshal(Config{
					Deployment: "/some/manifest.yml",
					Stemcells:  stemcells,
				})
				fakeFs.WriteFile(configFilePath, configFileContents)
				fakeFs.WriteFileString(deploymentFilePath, "some invalid content")
				config, err := service.Load()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshalling deployment file `/some/deployment.json'"))
				Expect(config).To(Equal(Config{}))
			})
		})
	})

	Describe("Save", func() {
		It("writes the given config to the config file", func() {
			config := Config{
				Deployment:     "/some/path",
				DeploymentUUID: "deadbeef",
				Stemcells:      stemcells,
			}

			err := service.Save(config)
			Expect(err).NotTo(HaveOccurred())

			configFileContents, err := fakeFs.ReadFileString(configFilePath)

			expectedConfig := Config{
				Deployment: "/some/path",
				Stemcells:  stemcells,
			}
			expectedConfigFileContents, err := json.MarshalIndent(expectedConfig, "", "    ")
			Expect(configFileContents).To(Equal(string(expectedConfigFileContents)))
		})

		It("writes the deployment uuid to the deployment file", func() {
			config := Config{
				Deployment:     "/some/manifest.yml",
				DeploymentUUID: "deadbeef",
				Stemcells:      stemcells,
			}

			err := service.Save(config)
			Expect(err).NotTo(HaveOccurred())

			deploymentFileContents, err := fakeFs.ReadFileString(deploymentFilePath)
			deploymentFile := DeploymentFile{
				UUID:      "deadbeef",
				Stemcells: stemcells,
			}
			expectedDeploymentFileContents, err := json.MarshalIndent(deploymentFile, "", "    ")
			Expect(deploymentFileContents).To(Equal(string(expectedDeploymentFileContents)))
		})

		Context("when the config file cannot be written", func() {
			BeforeEach(func() {
				fakeFs.WriteToFileError = errors.New("")
			})

			It("returns an error when it cannot write the config file", func() {
				config := Config{
					Deployment: "/some/path",
					Stemcells:  stemcells,
				}
				err := service.Save(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Writing config file"))
			})
		})

		XContext("when the deployment file cannot be written", func() {
			BeforeEach(func() {
			})

			It("returns an error when it cannot write the config file", func() {
				config := Config{
					Deployment: "/some/path",
					Stemcells:  stemcells,
				}
				err := service.Save(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Writing deployment file"))
			})
		})
	})
})