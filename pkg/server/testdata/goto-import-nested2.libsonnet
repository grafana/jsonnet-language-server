local base = import 'goto-import-nested1.libsonnet';

base {
  api+:: {
    v1+:: {
      other_obj+:: {},
    },
  },
}
