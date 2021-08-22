package api

//
//// QueueResumeAction used to resume specify paused queue
////➜  ~ curl  --user user:password -XPOST http://localhost:2900/_proxy/queue/resume -d'{"queue":"primary"}'
//func (handler *API) QueueResumeAction(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
//
//	o, err := handler.GetJSON(req)
//	if err != nil {
//		handler.Error500(w, err.Error())
//		return
//	}
//
//	k, err := o.String("queue")
//	if err != nil {
//		handler.Error500(w, err.Error())
//		return
//	}
//
//	queue.ResumeRead(k)
//
//	handler.WriteJSON(w, util.MapStr{
//		"acknowledge": true,
//	}, 200)
//}
