import groovy.json.JsonSlurper
import org.apache.camel.catalog.DefaultCamelCatalog
import org.yaml.snakeyaml.Yaml
import org.yaml.snakeyaml.DumperOptions

def slurper = new JsonSlurper()
def catalog = new DefaultCamelCatalog()


def output = new TreeMap()
output['version'] = catalog.loadedVersion
output['components'] = [:]

catalog.findComponentNames().sort().each { name ->
    def json = slurper.parseText(catalog.componentJSonSchema(name))

    output['components'][name] = [:]
    output['components'][name]['dependency'] = [:]
    output['components'][name]['dependency']['groupId'] = json.component.groupId
    output['components'][name]['dependency']['artifactId'] = json.component.artifactId
    output['components'][name]['dependency']['version'] = json.component.version
    output['components'][name]['schemes'] = [ json.component.scheme.trim() ]
    if (json.component.alternativeSchemes) {
        json.component.alternativeSchemes.split(',').each {
            scheme -> output['components'][name]['schemes'] << scheme.trim()
        }
    }
}

def options = new DumperOptions()
options.indent = 2
options.defaultFlowStyle = DumperOptions.FlowStyle.BLOCK

new File(catalogOutputFile).newWriter().withWriter {
    w -> w << new Yaml(options).dump(output)
}