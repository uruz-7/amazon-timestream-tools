package com.amazonaws.services.timestream;

import org.kohsuke.args4j.CmdLineException;
import org.kohsuke.args4j.CmdLineParser;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

public class Main {

    private static final Logger LOGGER = LoggerFactory.getLogger(Main.class);

    public static void main(final String[] args) throws IOException {
        final InputArguments inputArguments = parseArguments(args);
        LOGGER.info(String.format("Database: %s, Host: %s, Endpoint: %s",
                inputArguments.database, inputArguments.host, inputArguments.endpoint));

        final QueryExample queryExample = new QueryExample(
                inputArguments.database, inputArguments.host);

        if (inputArguments.endpoint) {
            queryExample.runAllQueriesWithEndpointConnection();
        } else {
            // Query samples using the JDBC driver.
            queryExample.runAllQueriesWithSimpleConnection();
        }

        System.exit(0);
    }

    private static InputArguments parseArguments(final String[] args) {
        final InputArguments inputArguments = new InputArguments();
        final CmdLineParser parser = new CmdLineParser(inputArguments);
        try {
            parser.parseArgument(args);
        } catch (final CmdLineException e) {
            System.err.println(e.getMessage());
            parser.printUsage(System.err);
            System.exit(1);
        }
        return inputArguments;
    }
}

