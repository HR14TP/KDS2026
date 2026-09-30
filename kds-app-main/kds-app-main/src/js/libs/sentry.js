'use strict';
const Sentry = require("@sentry/electron");

const kds_sentry = {
    name: 'kds_sentry',
    id: 'kds_sentry',
    args: [],
    init: () => {
        Sentry.init({ dsn: "https://2c9cbbe0dfd3411099ead4f8fc781bfc@o890609.ingest.sentry.io/5839628" });
    },
};

module.exports = kds_sentry;