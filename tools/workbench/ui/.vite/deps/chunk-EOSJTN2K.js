import {
  t4Templating
} from "./chunk-WFAS7KHP.js";
import {
  csharp
} from "./chunk-VGZQFQKZ.js";

// node_modules/refractor/lang/t4-cs.js
t4Cs.displayName = "t4-cs";
t4Cs.aliases = ["t4"];
function t4Cs(Prism) {
  Prism.register(csharp);
  Prism.register(t4Templating);
  Prism.languages.t4 = Prism.languages["t4-cs"] = Prism.languages["t4-templating"].createT4("csharp");
}

export {
  t4Cs
};
//# sourceMappingURL=chunk-EOSJTN2K.js.map
