package tests

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExport_ExportSimple(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_")
	if err != nil {
		t.Fatal("Cannot make temp dir:", err)
	}
	defer os.RemoveAll(dir)

	err = runRockerBuild(`
		FROM alpine:latest
		RUN echo -n "test_export" > /exported_file
		EXPORT /exported_file

		FROM alpine:latest
		MOUNT `+dir+`:/datadir
		IMPORT /exported_file /datadir/imported_file`, "--no-cache")
	if err != nil {
		t.Fatal(err)
	}

	content, err := ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}

	assert.Equal(t, "test_export", string(content))
}

func TestExport_ExportSeparateFilesDifferentExport(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_")
	if err != nil {
		t.Fatal("Cannot create tmp dir:", err)
	}
	defer os.RemoveAll(dir)

	rockerContentFirst := `FROM alpine:latest
						   EXPORT /etc/hostname
						   RUN echo -n "first_diff" > /exported_file
						   EXPORT /exported_file

						   FROM alpine
						   MOUNT ` + dir + `:/datadir
						   IMPORT /exported_file /datadir/imported_file`

	rockerContentSecond := `FROM alpine:latest
						    EXPORT /etc/issue
						    RUN echo -n "second_diff" > /exported_file
						    EXPORT /exported_file 

						    FROM alpine
						    MOUNT ` + dir + `:/datadir
						    IMPORT /exported_file /datadir/imported_file`

	err = runRockerBuild(rockerContentFirst, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = runRockerBuild(rockerContentSecond, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = runRockerBuild(rockerContentFirst)
	if err != nil {
		t.Fatal(err)
	}

	content, err := ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}

	assert.Equal(t, "first_diff", string(content))
}

func TestExport_ExportSmolinIssue(t *testing.T) {
	tag := "rocker-integratin-test-export-smolin"
	defer removeImage(tag + ":qa")
	defer removeImage(tag + ":prod")

	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_smolin")
	if err != nil {
		t.Fatal("Can't create tmp dir", err)
	}
	defer os.RemoveAll(dir)

	rockerfile, err := createTempFile("")
	if err != nil {
		t.Fatal("Can't create temp file", err)
	}
	defer os.RemoveAll(rockerfile)
	randomData := strconv.Itoa(int(time.Now().UnixNano() % int64(100000001)))

	rockerContentFirst := []byte(` {{ $env := .env}}
							 FROM alpine
							 RUN echo -n "{{ $env }}" > /exported_file
						 	 EXPORT /exported_file

							 FROM alpine
							 IMPORT /exported_file /imported_file
							 TAG ` + tag + ":{{ $env }}")

	rockerContentSecond := []byte(` {{ $env := .env}}
							 FROM alpine
							 RUN echo -n "{{ $env }}" > /exported_file
						 	 EXPORT /exported_file

							 FROM alpine
							 RUN echo "invalidate with ` + randomData + `"
							 IMPORT /exported_file /imported_file
							 TAG ` + tag + ":{{ $env }}")

	err = ioutil.WriteFile(rockerfile, rockerContentFirst, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--reload-cache", "--var", "env=qa")
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(rockerfile, rockerContentFirst, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--reload-cache", "--var", "env=prod")
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(rockerfile, rockerContentSecond, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--var", "env=qa")
	if err != nil {
		t.Fatal(err)
	}

	content := `FROM ` + tag + `:qa
					   MOUNT ` + dir + `:/data
					   RUN cp /imported_file /data/qa.file

					   FROM ` + tag + `:prod
					   MOUNT ` + dir + `:/data
					   RUN cp /imported_file /data/prod.file`
	err = runRockerBuild(content, "--no-cache")
	if err != nil {
		t.Fatal(err)
	}

	qaContent, err := ioutil.ReadFile(dir + "/qa.file")
	if err != nil {
		t.Fatal("failed to read qa.file:", err)
	}
	assert.Equal(t, string(qaContent), "qa")

	prodContent, err := ioutil.ReadFile(dir + "/prod.file")
	if err != nil {
		t.Fatal("failed to read prod.file:", err)
	}
	assert.Equal(t, string(prodContent), "prod")

}
func TestExport_ExportSeparateFilesSameExport(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_sep")
	if err != nil {
		t.Fatal("Can't create tmp dir", err)
	}
	defer os.RemoveAll(dir)

	rockerContentFirst := `FROM alpine:latest
						   EXPORT /etc/issue
						   RUN echo -n "first_separate" > /exported_file
						   EXPORT /exported_file

						   FROM alpine
						   MOUNT ` + dir + `:/datadir
						   IMPORT /exported_file /datadir/imported_file
						   `

	rockerContentSecond := `FROM alpine:latest
						    EXPORT /etc/issue
						    RUN echo -n "second_separate" > /exported_file
						    EXPORT /exported_file

						    FROM alpine
						    MOUNT ` + dir + `:/datadir
						    IMPORT /exported_file /datadir/imported_file`

	rockerContentThird := `FROM alpine:latest
						   EXPORT /etc/issue
						   RUN echo -n "first_separate" > /exported_file
						   EXPORT /exported_file

						   FROM alpine
						   MOUNT ` + dir + `:/datadir
						   RUN echo "Invalidate cache"
						   IMPORT /exported_file /datadir/imported_file
						   `

	err = runRockerBuild(rockerContentFirst, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = runRockerBuild(rockerContentSecond)
	if err != nil {
		t.Fatal(err)
	}

	err = runRockerBuild(rockerContentThird)
	if err != nil {
		t.Fatal(err)
	}

	content, err := ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}

	assert.Equal(t, "first_separate", string(content))
}

func TestExport_ExportSameFileDifferentCmd(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_")
	if err != nil {
		t.Fatal("Can't create tmp dir", err)
	}
	defer os.RemoveAll(dir)

	rockerfile, err := createTempFile("")
	if err != nil {
		t.Fatal("Can't create temp file", err)
	}
	defer os.RemoveAll(rockerfile)

	rockerContentFirst := []byte(`FROM alpine
						 	 RUN echo -n "first_foobar1" > /exported_file
						 	 EXPORT /exported_file
						 	 FROM alpine
						 	 MOUNT ` + dir + `:/datadir
						 	 IMPORT /exported_file /datadir/imported_file`)

	rockerContentSecond := []byte(`FROM alpine
							  RUN echo -n "second_foobar1" > /exported_file
							  EXPORT /exported_file
							  FROM alpine
							  MOUNT ` + dir + `:/datadir
							  IMPORT /exported_file /datadir/imported_file`)

	rockerContentThird := []byte(`FROM alpine
						 	 RUN echo -n "first_foobar1" > /exported_file
						 	 EXPORT /exported_file
						 	 FROM alpine
						 	 MOUNT ` + dir + `:/datadir
							 RUN echo "Invalidate cache"
						 	 IMPORT /exported_file /datadir/imported_file`)

	err = ioutil.WriteFile(rockerfile, rockerContentFirst, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}
	content, err := ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}
	assert.Equal(t, "first_foobar1", string(content))

	err = ioutil.WriteFile(rockerfile, rockerContentSecond, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile)
	if err != nil {
		t.Fatal(err)
	}
	content, err = ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}
	assert.Equal(t, "second_foobar1", string(content))

	err = ioutil.WriteFile(rockerfile, rockerContentThird, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile)
	if err != nil {
		t.Fatal(err)
	}
	content, err = ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}
	assert.Equal(t, "first_foobar1", string(content))
}

func TestExport_ExportSameFileFewFroms(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "rocker_integration_test_export_")
	if err != nil {
		t.Fatal("Can't create tmp dir", err)
	}
	defer os.RemoveAll(dir)

	rockerfile, err := createTempFile("")
	if err != nil {
		t.Fatal("Can't create temp file", err)
	}
	defer os.RemoveAll(rockerfile)

	rockerContentFirst := []byte(`FROM alpine
								  EXPORT /etc/issue

								  FROM alpine
								  RUN echo -n "first_few" > /exported_file
								  EXPORT /exported_file

						 	      FROM alpine
						 	      MOUNT ` + dir + `:/datadir
						 	      IMPORT /exported_file /datadir/imported_file`)

	rockerContentSecond := []byte(`FROM alpine
								  EXPORT /etc/issue

								  FROM alpine
								  RUN echo -n "second_few" > /exported_file
								  EXPORT /exported_file`)

	err = ioutil.WriteFile(rockerfile, rockerContentFirst, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(rockerfile, rockerContentSecond, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(rockerfile, rockerContentFirst, 0644)
	if err != nil {
		t.Fatal("failed to write Rockerfile:", err)
	}
	err = runRockerBuildWithFile(rockerfile)
	if err != nil {
		t.Fatal(err)
	}

	content, err := ioutil.ReadFile(dir + "/imported_file")
	if err != nil {
		t.Fatal("Cannot read imported_file:", err)
	}

	assert.Equal(t, "first_few", string(content))
}

func TestExport_DoubleExport(t *testing.T) {
	rockerContent := `FROM alpine
					  EXPORT /etc/issue issue
					  EXPORT /etc/hostname hostname

					  FROM alpine
					  IMPORT issue
					  IMPORT hostname`

	err := runRockerBuild(rockerContent, "--reload-cache")
	if err != nil {
		t.Fatal(err)
	}

	err = runRockerBuild(rockerContent)
	if err != nil {
		t.Fatal(err)
	}
}
