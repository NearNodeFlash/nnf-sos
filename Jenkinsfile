// Procedure for building NNF Storage Orchestration Services

@Library('dst-shared@master') _

// See https://github.hpe.com/hpe/hpc-dst-jenkins-shared-library for all
// the inputs to the dockerBuildPipeline.
// In particular: vars/dockerBuildPipeline.groovy
dockerBuildPipeline {
        repository = "cray"
        imagePrefix = "cray"
        app = "dp-nnf-sos"
        name = "dp-nnf-sos"
        description = "Near Node Flash Storage Orchestration Services"
        dockerfile = "Dockerfile"
        autoJira = false
        createSDPManifest = false
        product = "rabsw"
}
