context {
    registry {
        myEntry1 = 'myRegistryEntry1'
        myEntry2 = 'myRegistryEntry2'
        myEntry3 = processor {
            it.in.headers['test'] = 'value'
        }
    }
}

from('timer:tick')
    .to('log:info')