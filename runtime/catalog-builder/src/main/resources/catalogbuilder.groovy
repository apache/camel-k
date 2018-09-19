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

    if (!output['artifacts'][id]['schemes'].contains(json.component.scheme.trim())) {
        output['artifacts'][id]['schemes'] << json.component.scheme.trim()
    }

    if (json.component.alternativeSchemes) {
        json.component.alternativeSchemes.split(',').collect {
            scheme -> scheme.trim()
        }.findAll {
            scheme -> !output['artifacts'][id]['schemes'].contains(scheme)
        }.each { 
            scheme -> output['artifacts'][id]['schemes'] << scheme
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