// node_modules/refractor/lang/hpkp.js
hpkp.displayName = "hpkp";
hpkp.aliases = [];
function hpkp(Prism) {
  Prism.languages.hpkp = {
    directive: {
      pattern: /\b(?:includeSubDomains|max-age|pin-sha256|preload|report-to|report-uri|strict)(?=[\s;=]|$)/i,
      alias: "property"
    },
    operator: /=/,
    punctuation: /;/
  };
}

export {
  hpkp
};
//# sourceMappingURL=chunk-4W3FP4T7.js.map
