import { defineStore } from "pinia";
import { useProvider } from "./provider";
import { ref, watch } from "vue";
import { useTerminal } from "./terminal";
import { isMessage } from "@bufbuild/protobuf";
import * as wasimoff from "@wasimoff/proto/v1/messages_pb";

export const useClusterState = defineStore("ClusterState", () => {
  const providerstore = useProvider();
  const terminal = useTerminal();

  // number of providers currently connected to the broker
  const providers = ref<number>();

  // current throughput of tasks per second
  const throughput = ref<number>(0);

  // whenever the provider messenger reconnects
  watch(
    () => providerstore.$messenger,
    async (messenger) => {
      if (messenger !== undefined && providerstore.$provider !== undefined) {
        // read messages from the event stream
        for await (const event of messenger.events) {
          switch (
            true // switch by message type
          ) {
            // print generic messages to the terminal
            case isMessage(event, wasimoff.Event_GenericMessageSchema):
              terminal.info(`Message: ${event.message}`);
              break;

            // update provider count
            case isMessage(event, wasimoff.Event_ClusterInfoSchema):
              providers.value = event.providers;
              break;

            // update throughput
            case isMessage(event, wasimoff.Event_ThroughputSchema):
              throughput.value = event.overall;
              break;
          }
        }

        // tidy up when disconnected
        providers.value = undefined;
      }
    },
  );

  return { providers, throughput };
});
