// Procedure for building NNF Element Controller  

@Library('dst-shared@master') _

dockerBuildPipeline {
        repository = "cray"
        imagePrefix = "cray"
        app = "dp-nnf-sos"
        name = "dp-nnf-sos"
        description = "Near Node Flash Storage Orchestration Services"
        dockerfile = "Dockerfile"
        useLazyDocker = true
        autoJira = false
        createSDPManifest = false
        product = "kj"
}
