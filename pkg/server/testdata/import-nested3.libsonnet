(import 'import-nested2.libsonnet')
+ {
  local this = self,
  _config+:: {
    some: true,
    attributes: this.util,
  },

  util+:: {
    // other stuff
  },
}
