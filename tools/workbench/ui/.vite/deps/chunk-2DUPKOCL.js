import {
  json
} from "./chunk-SS4PDKSE.js";

// node_modules/refractor/lang/jsonp.js
jsonp.displayName = "jsonp";
jsonp.aliases = [];
function jsonp(Prism) {
  Prism.register(json);
  Prism.languages.jsonp = Prism.languages.extend("json", {
    punctuation: /[{}[\]();,.]/
  });
  Prism.languages.insertBefore("jsonp", "punctuation", {
    function: /(?!\s)[_$a-zA-Z\xA0-\uFFFF](?:(?!\s)[$\w\xA0-\uFFFF])*(?=\s*\()/
  });
}

export {
  jsonp
};
//# sourceMappingURL=chunk-2DUPKOCL.js.map
