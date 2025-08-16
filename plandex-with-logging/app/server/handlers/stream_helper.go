package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	modelPlan "plandex-server/model/plan"
	"plandex-server/types"
	"time"

	shared "plandex-shared"
)

const HeartbeatInterval = 5 * time.Second

func startResponseStream(reqCtx context.Context, w http.ResponseWriter, auth *types.ServerAuth, planId, branch string, isConnect bool) {
	log.Println("Response stream manager: starting plan stream")

	active := modelPlan.GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("Response stream manager: active plan not found for plan ID %s on branch %s\n", planId, branch)
		http.Error(w, "Active plan not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// send initial message to client
	msg := shared.StreamMessage{
		Type: shared.StreamMessageStart,
	}

	bytes, err := json.Marshal(msg)

	if err != nil {
		log.Printf("Response stream manager: error marshalling message: %v\n", err)
		return
	}

	log.Println("Response stream manager: sending initial message")
	err = sendStreamMessage(w, string(bytes))
	if err != nil {
		log.Println("Response stream manager: error sending initial message:", err)
		return
	}

	if isConnect {
		time.Sleep(100 * time.Millisecond)
		err = initConnectActive(auth, planId, branch, w)

		if err != nil {
			log.Println("Response stream manager: error initializing connection to active plan:", err)
			return
		}
	}

	subscriptionId, ch := modelPlan.SubscribePlan(reqCtx, planId, branch)
	defer func() {
		log.Println("Response stream manager: client stream closed")
		modelPlan.UnsubscribePlan(planId, branch, subscriptionId)
	}()

	if isConnect {
		time.Sleep(50 * time.Millisecond)
	} else {
		time.Sleep(100 * time.Millisecond)
	}

	chHeartbeat := make(chan string)

	// send heartbeats while the stream is active
	go func() {
		ticker := time.NewTicker(HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				chHeartbeat <- string(shared.StreamMessageHeartbeat)
			case <-reqCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-reqCtx.Done():
			log.Println("Response stream manager: request context done")
			return
		case msg := <-chHeartbeat:
			err = sendStreamMessage(w, msg)
			if err != nil {
				return
			}
		case msg := <-ch:
			// log.Println("Response stream manager: sending message:", msg)
			err = sendStreamMessage(w, msg)
			if err != nil {
				return
			}
		}
	}

}

func sendStreamMessage(w http.ResponseWriter, msg string) error {
	bytes := []byte(msg + shared.STREAM_MESSAGE_SEPARATOR)

	// log.Printf("Response stream manager: writing message to client: %s\n", msg)

	_, err := w.Write(bytes)
	if err != nil {
		log.Printf("Response stream manager: error writing to client: %v\n", err)
		return err
	} else if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func initConnectActive(auth *types.ServerAuth, planId, branch string, w http.ResponseWriter) error {
	log.Println("Response stream manager: initializing connection to active plan")

	active := modelPlan.GetActivePlan(planId, branch)

	if active == nil {
		return fmt.Errorf("active plan not found for plan ID %s on branch %s", planId, branch)
	}

	msg := shared.StreamMessage{
		Type: shared.StreamMessageConnectActive,
	}

	if active.Prompt != "" && !active.BuildOnly {
		msg.InitPrompt = active.Prompt
	}

	if active.BuildOnly {
		msg.InitBuildOnly = true
	}

	if len(active.StoredReplyIds) > 0 {
		convo, err := db.GetPlanConvo(auth.OrgId, active.Id)
		if err != nil {
			return fmt.Errorf("error getting plan convo: %v", err)
		}

		convoMsgById := map[string]*db.ConvoMessage{}
		for _, convoMsg := range convo {
			convoMsgById[convoMsg.Id] = convoMsg
		}

		for _, replyId := range active.StoredReplyIds {
			if convoMsg, ok := convoMsgById[replyId]; ok {
				msg.InitReplies = append(msg.InitReplies, convoMsg.Message)
			}
		}
	}

	if active.CurrentReplyContent != "" {
		msg.InitReplies = append(msg.InitReplies, active.CurrentReplyContent)
	}

	if active.MissingFilePath != "" {
		msg.MissingFilePath = active.MissingFilePath
	}

	bytes, err := json.Marshal(msg)

	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}

	log.Println("Response stream manager: sending connect message")
	err = sendStreamMessage(w, string(bytes))

	if err != nil {
		return fmt.Errorf("error sending connect message: %v", err)
	}

	buildQueuesByPath := modelPlan.GetActivePlan(planId, branch).BuildQueuesByPath

	// if we're connecting to an active stream and there are active builds, send initial build info
	if len(buildQueuesByPath) > 0 {

		for path, queue := range buildQueuesByPath {
			buildInfo := shared.BuildInfo{Path: path}

			for _, build := range queue {
				if build.BuildFinished() {
					buildInfo.NumTokens = 0
					buildInfo.Finished = true
				} else {
					// no longer showing token counts in build info - leaving commented out for now for reference
					// tokens := build.WithLineNumsBufferTokens
					buildInfo.Finished = false
					// buildInfo.NumTokens += tokens
				}
			}

			msg := shared.StreamMessage{
				Type:      shared.StreamMessageBuildInfo,
				BuildInfo: &buildInfo,
			}
			bytes, err := json.Marshal(msg)

			if err != nil {
				return fmt.Errorf("error marshalling message: %v", err)
			}

			err = sendStreamMessage(w, string(bytes))

			if err != nil {
				return fmt.Errorf("error sending message: %v", err)
			}

		}

	}

	return nil
}
