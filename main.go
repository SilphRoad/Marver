/*
Marver
Copyright (C) 2018  Jacob Wiltse

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"strconv"
)

func main() {
	Token := os.Getenv("TOKEN")
	webhookID := os.Getenv("WEBHOOK_ID")
	webhookToken := os.Getenv("WEBHOOK_TOKEN")
	namespace := os.Getenv("NAMESPACE")
	statefulSetName := os.Getenv("STATEFULSET")
	containerName := os.Getenv("CONTAINER_NAME")
	envVar := os.Getenv("COUNT_ENV_VAR")
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		panic(err)
	}
	st, err := dg.GatewayBot()
	if err != nil {
		panic(err)
	}
	fmt.Println("Recommended Shards:", st.Shards)
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	statefulSetsClient := clientSet.AppsV1().StatefulSets(namespace)

	result, err := statefulSetsClient.Get(statefulSetName, metav1.GetOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to get StatefulSet: %v", err))
	}
	shardCount := int32(st.Shards)
	if shardCount > *result.Spec.Replicas {
		// Set the replica count
		result.Spec.Replicas = &shardCount
		// Set the shard count environment variable - will force existing shards to restart
		for i, container := range result.Spec.Template.Spec.Containers {
			if container.Name == containerName {
				for j, environmentVariable := range result.Spec.Template.Spec.Containers[i].Env {
					if environmentVariable.Name == envVar {
						result.Spec.Template.Spec.Containers[i].Env[j].Value = strconv.Itoa(st.Shards)
					}
				}
			}
		}
		_, err = statefulSetsClient.Update(result)
		fmt.Println("Updated StatefulSet to", st.Shards, "replicas")
		params := discordgo.WebhookParams{
			Content:  fmt.Sprintf("Bot has been scaled to %d shards.", st.Shards),
			Username: "Marver",
		}
		_, err = dg.WebhookExecute(
			webhookID,
			webhookToken,
			false,
			&params)
		if err != nil {
			panic(err)
		}
	}
}
