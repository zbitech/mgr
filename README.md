# ZBI Resource Template Manager Service
The resource template manager service is a go-lang package that defines the 
representation of projects and instances in the repository and in Kubernetes. 
For the purposes of this project, the control plane retains overall ownership 
of these resources within the Kubernetes environment and uses the repository to 
store the meta-data necessary to represent platform-level user ownership and team 
association.

It also provides representations of the required Kubernetes resources for running 
a Zcash node instance. These resources include configuration files, application 
credentials, storage, and the node software binary. These resources are represented 
as template files from which the target application can be generated. This allows 
for each deployed node to follow an optimal configuration while maintaining its 
independence within adefined boundary. Additionally, it can be extended to support 
any compatible Zcash node by providing a corresponding resource template manager.

### Dependencies
- http://github.com/zbitech/common
- https://pkg.go.dev/text/template

## Project Manager

## Instance Managers

### Zcash Manager

### Lightwalletd Server Manager