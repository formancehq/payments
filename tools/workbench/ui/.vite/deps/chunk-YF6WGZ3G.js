import {
  php
} from "./chunk-427W6S3R.js";

// node_modules/refractor/lang/php-extras.js
phpExtras.displayName = "php-extras";
phpExtras.aliases = [];
function phpExtras(Prism) {
  Prism.register(php);
  Prism.languages.insertBefore("php", "variable", {
    this: {
      pattern: /\$this\b/,
      alias: "keyword"
    },
    global: /\$(?:GLOBALS|HTTP_RAW_POST_DATA|_(?:COOKIE|ENV|FILES|GET|POST|REQUEST|SERVER|SESSION)|argc|argv|http_response_header|php_errormsg)\b/,
    scope: {
      pattern: /\b[\w\\]+::/,
      inside: {
        keyword: /\b(?:parent|self|static)\b/,
        punctuation: /::|\\/
      }
    }
  });
}

export {
  phpExtras
};
//# sourceMappingURL=chunk-YF6WGZ3G.js.map
