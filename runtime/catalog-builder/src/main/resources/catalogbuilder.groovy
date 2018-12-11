import groovy.json.JsonSlurper
import org.apache.camel.catalog.DefaultCamelCatalog
import org.yaml.snakeyaml.Yaml
import org.yaml.snakeyaml.DumperOptions

def slurper = new JsonSlurper()
def catalog = new DefaultCamelCatalog()


def output = new TreeMap()
output['version'] = catalog.loadedVersion
output['artifacts'] = [:]



// *******************************
//
// https://github.com/apache/camel-k/issues/264
//
// *******************************

def httpURIs = [
	"ahc",
	"ahc-ws",
	"atmosphere-websocket",
	"cxf",
	"cxfrs",
	"grpc",
	"jetty",
	"netty-http",
	"netty4-http",
	"rest",
	"restlet",
	"servlet",
	"spark-rest",
	"spring-ws",
	"undertow",
	"websocket",
	"knative"
]

def passiveURIs = [
	"bean",
	"binding",
	"browse",
	"class",
	"controlbus",
	"dataformat",
	"dataset",
	"direct",
	"direct-vm",
	"language",
	"log",
	"mock",
	"properties",
	"ref",
	"seda",
	"stub",
	"test",
	"validator",
	"vm"
]

// *******************************
//
// Components
//
// *******************************

catalog.findComponentNames().sort().each { name ->
    def json = slurper.parseText(catalog.componentJSonSchema(name))
    def id = json.component.artifactId

    if (!output['artifacts'].containsKey(id)) {
        output['artifacts'][id] = [:]
        output['artifacts'][id]['groupId'] = json.component.groupId
        output['artifacts'][id]['artifactId'] = json.component.artifactId
        output['artifacts'][id]['version'] = json.component.version
        output['artifacts'][id]['schemes'] = []
        output['artifacts'][id]['languages'] = []
        output['artifacts'][id]['dataformats'] = []
    }

    def schemes = output['artifacts'][id]['schemes']
    def scheme = json.component.scheme.trim()

    if (!schemes.any{ it['id'] == scheme}) {
        schemes << [ id: scheme ]
    }

    if (json.component.alternativeSchemes) {
        json.component.alternativeSchemes.split(',').collect {
            it.trim()
        }.findAll {
            !schemes.any{ it['id'] == scheme }
        }.each { 
            schemes << [ id: scheme ]
        }
    }

    schemes?.each {
        if (httpURIs.contains(it['id'])) {
            it['http'] = true
        }
        if (passiveURIs.contains(it['id'])) {
            it['passive'] = true
        }
    }
}

// *******************************
//
// Languages
//
// *******************************

catalog.findLanguageNames().sort().each { name ->
    def json = slurper.parseText(catalog.languageJSonSchema(name))
    def id = json.language.artifactId

    if (!output['artifacts'].containsKey(id)) {
        output['artifacts'][id] = [:]
        output['artifacts'][id]['groupId'] = json.language.groupId
        output['artifacts'][id]['artifactId'] = json.language.artifactId
        output['artifacts'][id]['version'] = json.language.version
        output['artifacts'][id]['components'] = []
        output['artifacts'][id]['languages'] = []
        output['artifacts'][id]['dataformats'] = []
    }

    if (!output['artifacts'][id]['languages'].contains(json.language.name)) {
        output['artifacts'][id]['languages'] << json.language.name
    }
}

// *******************************
//
// Dataformat
//
// *******************************

catalog.findDataFormatNames().sort().each { name ->
    def json = slurper.parseText(catalog.dataFormatJSonSchema(name))
    def id = json.dataformat.artifactId

    if (!output['artifacts'].containsKey(id)) {
        output['artifacts'][id] = [:]
        output['artifacts'][id]['groupId'] = json.dataformat.groupId
        output['artifacts'][id]['artifactId'] = json.dataformat.artifactId
        output['artifacts'][id]['version'] = json.dataformat.version
        output['artifacts'][id]['components'] = []
        output['artifacts'][id]['languages'] = []
        output['artifacts'][id]['dataformats'] = []
    }

    if (!output['artifacts'][id]['dataformats'].contains(json.dataformat.name)) {
        output['artifacts'][id]['dataformats'] << json.dataformat.name
    }
}

// *******************************
//
// 
//
// *******************************

def options = new DumperOptions()
options.indent = 2
options.defaultFlowStyle = DumperOptions.FlowStyle.BLOCK

new File(catalogOutputFile).newWriter().withWriter {
    w -> w << new Yaml(options).dump(output)
}